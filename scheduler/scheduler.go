package scheduler

import (
	"summerWebCrawler/analyzer"
	"summerWebCrawler/itempipeline"
	"net/http"
	"summerWebCrawler/middleware"
	"summerWebCrawler/downloadder"
	"fmt"
	"errors"
	"summerWebCrawler/logging"
	"summerWebCrawler/base"
	"sync/atomic"
	"time"
	"strings"
)

//调度器的接口类型
type Scheduler interface {
	//启动调度器
	//调用该方法会使调度器创建和初始化各个组件.在此之后,调度器会激活爬取流程的执行
	//参数channelLen被用来指定数据传输通道的长度
	//参数poolSize被用来设定网页下载池和分析器池的容量
	//参数crawlDepth代表了需要被爬取的网页的最大深度值,深度大于此值的网页会被忽略
	//参数httpClicentGenerator代表的是被用来生成http客户端的函数
	//参数respParsers的值应为需要被置入条目处理管道中的条目处理器的序列
	//参数firstHttpReq即代表首次请求.调度器会以此为起点开始执行爬取流程
	Start(args base.ChannelArgs,
		baseArgs base.PoolBaseArgs,
		crawlDepth uint32,
		httpClientGenerator GenHttpClient,
		respParsers []analyzer.ParseResponse,
		itemProcessors []itempipeline.ProcessItem,
		firstHttpReq *http.Request) (err error)

	//调用该方法会停止调度器的运行.所有处理模块执行的流程会被中止
	Stop() bool
	//判断调度器是否正在运行
	Running() bool
	//获得错误通道,调度器以及各个处理模块运行过程中出现的所有错误都会被发送到该通道
	ErrorChan() <-chan error
	//获取所有处理模块是否都处于空闲状态
	Idle() bool
	//获取摘要信息
	Summary(prefix string) SchedSummary
}

//被用来生成http客户端的函数类型
type GenHttpClient func() *http.Client

//调度器的实现
type myScheduler struct {
	//池的尺寸
	poolSizeArgs base.PoolBaseArgs
	//通道的长度(也即容量)
	channelArgs base.ChannelArgs
	//爬取的最大深度,首次氢气的深度为0
	crawlDepth uint32
	//主域名
	primaryDomain string

	//通道管理器
	chanman middleware.ChannelManager
	//停止信号
	stopSign middleware.StopSign
	//网页下载器池
	dlPool downloadder.PageDownloaderPool
	//分析器池
	analyzerPool analyzer.AnalyzerPool
	//条目处理管道
	itemPipeline itempipeline.ItemPipeline

	//运行标记,0表示未运行,1表示已运行,2表示已停止
	running uint32
	//请求缓存
	reqCache requestCache
	//已请求的URL的字典
	urlMap map[string]bool
}

// 日志记录器。
var logger logging.Logger = base.NewLogger()

const (
	DOWNLOADER_CODE   = "downloader"
	ANALYZER_CODE     = "analyzer"
	ITEMPIPELINE_CODE = "item_pipeline"
	SCHEDULER_CODE    = "scheduler"
)

//创建调度器
func NewScheduler() Scheduler {
	return &myScheduler{}
}

func (scheduler *myScheduler) Start(channelArgs base.ChannelArgs,
	poolSizeArgs base.PoolBaseArgs,
	crawlDepth uint32,
	httpClientGenerator GenHttpClient,
	respParsers []analyzer.ParseResponse,
	itemProcessors []itempipeline.ProcessItem,
	firstHttpReq *http.Request) (err error) {
	//初始化调度器的各个字段以及开启调度器的过程中有运行时的panic被抛出
	//调度器能够及时地恢复它并记录下相应的日志
	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal Scheduler Error:%s\n", p)
			logger.Fatal(errMsg)
			err = errors.New(errMsg)
		}
	}()
	//检查running字段.查看调度器的状态
	if atomic.LoadUint32(&scheduler.running) == 1 {
		return errors.New("The scheduler has been started!\n")
	}
	//更改调度器状态
	atomic.StoreUint32(&scheduler.running, 1)

	//检查channel的参数是否合法
	if err := channelArgs.Check(); err != nil {
		return err
	}
	scheduler.channelArgs = channelArgs

	//检查资源池的参数是否合法
	if err := poolSizeArgs.Check(); err != nil {
		return err
	}
	scheduler.poolSizeArgs = poolSizeArgs

	scheduler.crawlDepth = crawlDepth
	scheduler.chanman = generateChannelManager(scheduler.channelArgs)
	if httpClientGenerator == nil {
		return errors.New("The http client generator list is invalid!")
	}
	dlPool, err := generatePageDownloaderPool(scheduler.poolSizeArgs.PageDownloaderPoolSize(), httpClientGenerator)
	if err != nil {
		errMsg := fmt.Sprintf("Occur error when get page downloader pool:%s\n", err)
		return errors.New(errMsg)
	}
	scheduler.dlPool = dlPool
	analyzerPool, err := generateAnalyzerPool(scheduler.poolSizeArgs.AnalyzerPoolSize())
	if err != nil {
		errMsg := fmt.Sprintf("Occur error when get analyzer pool:%s\n", err)
		return errors.New(errMsg)
	}
	scheduler.analyzerPool = analyzerPool

	if itemProcessors == nil {
		return errors.New("The item processor list is invalid!")
	}

	for i, ip := range itemProcessors {
		if ip == nil {
			return errors.New(fmt.Sprintf("The %dth item processor is invalid!", i))
		}
	}
	scheduler.itemPipeline = generateItemPipeline(itemProcessors)

	if scheduler.stopSign == nil {
		scheduler.stopSign = middleware.NewStopSign()
	} else {
		scheduler.stopSign.Reset()
	}

	scheduler.reqCache = NewRequestCache()
	scheduler.urlMap = make(map[string]bool)

	scheduler.startDownloading()
	scheduler.activateAnalyzers(respParsers)
	scheduler.openItemPipeline()
	scheduler.schedule(10 * time.Millisecond)

	if firstHttpReq == nil {
		return errors.New("The first http request is invalid!")
	}
	pd, err := getPrimaryDomain(firstHttpReq.Host)
	if err != nil {
		return err
	}
	scheduler.primaryDomain = pd

	firstreq := base.NewRequest(firstHttpReq, 0)
	scheduler.reqCache.put(firstreq)

	return nil
}

//开始下载
func (scheduler *myScheduler) startDownloading() {
	go func() {
		for {
			req, ok := <-scheduler.getReqChan()
			if !ok {
				break
			}
			go scheduler.download(req)
		}
	}()
}

//获取通道管理器持有的请求channel
func (scheduler *myScheduler) getReqChan() chan base.Request {
	reqChan, err := scheduler.chanman.ReqChan()
	if err != nil {
		panic(err)
	}
	return reqChan
}

//激活分析器
func (scheduler *myScheduler) activateAnalyzers(respParsers []analyzer.ParseResponse) {
	go func() {
		for {
			resp, ok := <-scheduler.getRespChan()
			if !ok {
				break
			}
			go scheduler.analyze(respParsers, resp)
		}
	}()
}

func (scheduler *myScheduler) getRespChan() chan base.Response {
	respChan, err := scheduler.chanman.RespChan()
	if err != nil {
		panic(err)
	}
	return respChan
}

func (scheduler *myScheduler) getErrorChan() chan error {
	errChan, err := scheduler.chanman.ErrorChan()
	if err != nil {
		panic(err)
	}
	return errChan
}

func (scheduler *myScheduler) getItemChan() chan base.Item {
	itemChan, err := scheduler.chanman.ItemChan()
	if err != nil {
		panic(err)
	}
	return itemChan
}

func (scheduler *myScheduler) sendError(err error, code string) bool {

	if scheduler.stopSign.Signed() {
		scheduler.stopSign.Deal(code)
		return false
	}

	if err == nil {
		return false
	}
	codePrefix := parseCode(code)[0]
	var errorType base.ErrorType
	switch codePrefix {
	case DOWNLOADER_CODE:
		errorType = base.DOWNLOADER_ERROR
	case ANALYZER_CODE:
		errorType = base.ANALYZER_ERROR
	case ITEMPIPELINE_CODE:
		errorType = base.ITEM_PROCESSOR_ERROR
	}
	cError := base.NewCrawlerError(errorType, err.Error())

	go func() {
		scheduler.getErrorChan() <- cError
	}()

	return true
}

func (scheduler *myScheduler) download(req base.Request) {

	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal Download Error:%s\n", p)
			logger.Fatal(errMsg)
		}
	}()
	//从网页下载池中取出一个下载实体
	downloader, err := scheduler.dlPool.Take()
	defer func() {
		//归还下载器
		err := scheduler.dlPool.Return(downloader)
		if err != nil {
			errMsg := fmt.Sprintf("Downloader pool error:%s", err)
			scheduler.sendError(errors.New(errMsg), SCHEDULER_CODE)
		}
	}()
	if err != nil {
		errMsg := fmt.Sprintf("Downloader pool error:%s", err)
		scheduler.sendError(errors.New(errMsg), SCHEDULER_CODE)
		return
	}
	code := generateCode(DOWNLOADER_CODE, downloader.Id())
	respp, err := downloader.Download(req)
	if respp != nil {
		scheduler.sendResp(*respp, code)
	}
	if err != nil {
		scheduler.sendError(err, code)
	}

}

func (scheduler *myScheduler) sendResp(resp base.Response, code string) bool {
	//判断是否已经停止
	if scheduler.stopSign.Signed() {
		scheduler.stopSign.Deal(code)
		return false
	}
	scheduler.getRespChan() <- resp
	return true
}

func (scheduler *myScheduler) sendItem(item base.Item, code string) bool {
	//判断程序是否已经停止
	if scheduler.stopSign.Signed() {
		scheduler.stopSign.Deal(code)
		return false
	}
	scheduler.getItemChan() <- item
	return true
}

func (scheduler *myScheduler) analyze(responses []analyzer.ParseResponse, response base.Response) {
	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal Analysis Error:%s\n", p)
			logger.Fatal(errMsg)
		}
	}()
	//从分析池取一个实体
	analyzer, err := scheduler.analyzerPool.Take()
	if err != nil {
		errMsg := fmt.Sprintf("Analyzer pool error:%s\n", err)
		scheduler.sendError(errors.New(errMsg), SCHEDULER_CODE)
		return
	}
	defer func() {
		err := scheduler.analyzerPool.Return(analyzer)
		if err != nil {
			errMsg := fmt.Sprintf("Analyzer pool error:%s\n", err)
			scheduler.sendError(errors.New(errMsg), SCHEDULER_CODE)
		}
	}()
	code := generateCode(ANALYZER_CODE, analyzer.Id())
	dataList, errs := analyzer.Analyze(responses, response)
	if dataList != nil {
		for _, data := range dataList {
			if data == nil {
				continue
			}
			switch d := data.(type) {
			case *base.Request:
				scheduler.saveReqToCache(*d, code)
			case *base.Item:
				scheduler.sendItem(*d, code)
			default:
				errMsg := fmt.Sprintf("Unsupported data type '%T'! (value=%v)\n", d, d)
				scheduler.sendError(errors.New(errMsg), code)
			}
		}
	}
	if errs != nil {
		for _, err := range errs {
			scheduler.sendError(err, code)
		}
	}
}

func (scheduler *myScheduler) saveReqToCache(request base.Request, code string) bool {
	httpReq := request.HttpReq()
	if httpReq == nil {
		logger.Warnln("Ignore the request! It's http request is invalid!")
		return false
	}
	reqUrl := httpReq.URL
	if reqUrl == nil {
		logger.Warnln("Ignore the request! It's url is invalid!")
		return false
	}
	if strings.ToLower(reqUrl.Scheme) != "http" {
		logger.Warnf("Ignore the request! It's url scheme '%s',but should be 'http'!\n", reqUrl.Scheme)
		return false
	}
	if _, ok := scheduler.urlMap[reqUrl.String()]; ok {
		logger.Warnf("Ignore the request! It's url is repeated. (reqeustUrl=%s)\n", reqUrl)
		return false
	}
	if pd, _ := getPrimaryDomain(reqUrl.Host); pd != scheduler.primaryDomain {
		logger.Warnf("Ignore the request!It's host '%s' not in primary domain '%s'. (requestUrl=%s)\n", httpReq.Host, scheduler.primaryDomain, reqUrl)
		return false
	}
	if request.Depth() > scheduler.crawlDepth {
		logger.Warnf("Ignore the request! It's depth %d greater than %d. (requestUrl=%s)", request.Depth(), scheduler.crawlDepth, reqUrl)
		return false
	}

	if scheduler.stopSign.Signed() {
		scheduler.stopSign.Deal(code)
		return false
	}
	scheduler.reqCache.put(&request)
	scheduler.urlMap[reqUrl.String()] = true
	return true
}

//打开条目处理管道
func (scheduler *myScheduler) openItemPipeline() {
	go func() {
		scheduler.itemPipeline.SetFailFast(true)
		code := ITEMPIPELINE_CODE
		for item := range scheduler.getItemChan() {
			go func(item base.Item) {
				defer func() {
					if p := recover(); p != nil {
						errMsg := fmt.Sprintf("Fatal Item Processing Error:%s\n", p)
						logger.Fatal(errMsg)
					}
					errs := scheduler.itemPipeline.Send(item)
					if errs != nil {
						for _, err := range errs {
							scheduler.sendError(err, code)
						}
					}
				}()
			}(item)
		}
	}()
}

//调度,适当的搬运请求缓存中的请求到请求通道
func (scheduler *myScheduler) schedule(interval time.Duration) {
	go func() {
		for {

			if scheduler.stopSign.Signed() {
				scheduler.stopSign.Deal(SCHEDULER_CODE)
				return
			}

			//计算请求channel的剩余空间作为调度依据
			remainder := cap(scheduler.getReqChan()) - len(scheduler.getReqChan())
			var temp *base.Request
			for remainder > 0 {
				temp = scheduler.reqCache.get()
				if temp == nil {
					break
				}
				//有必要多判断一次,因为程序可能时刻中断,
				// 而for循环内执行代码需要一定时间
				//所以在请求发送之前是有必要多判断一次的
				if scheduler.stopSign.Signed() {
					scheduler.stopSign.Deal(SCHEDULER_CODE)
					return
				}
				scheduler.getReqChan() <- *temp
				remainder--
			}
			time.Sleep(interval)
		}
	}()
}

func (scheduler *myScheduler) Stop() bool {
	if atomic.LoadUint32(&scheduler.running) != 1 {
		return false
	}

	scheduler.stopSign.Sign()
	scheduler.chanman.Close()
	scheduler.reqCache.close()
	atomic.StoreUint32(&scheduler.running, 2)
	return true
}

func (scheduler *myScheduler) Running() bool {
	return atomic.LoadUint32(&scheduler.running) == 1
}

func (scheduler *myScheduler) ErrorChan() <-chan error {
	if scheduler.chanman.Status() != middleware.CHANNEL_MANAGER_STATUS_INITIALIZED {
		return nil
	}
	return scheduler.getErrorChan()
}

//检查是否空闲
func (scheduler *myScheduler) Idle() bool {
	idleDlPool := scheduler.dlPool.Used() == 0
	idleAnalyzerPool := scheduler.analyzerPool.Used() == 0
	idleItemPipeline := scheduler.itemPipeline.ProcessingNumber() == 0

	if idleDlPool && idleAnalyzerPool && idleItemPipeline {
		return true
	}
	return false
}

//获取摘要信息
func (scheduler *myScheduler) Summary(prefix string) SchedSummary {
	return NewSchedSummary(scheduler, prefix)
}
