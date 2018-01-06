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
	Start(channelLen uint,
		poolSize uint32,
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

type SchedSummary interface {
	//获得摘要信息的一般表示
	String() string
	//获得摘要信息的详细表示
	Detail() string
	//判断是否与另一份摘要信息相同
	Same(other SchedSummary) bool
}

//被用来生成http客户端的函数类型
type GenHttpClient func() *http.Client

//调度器的实现
type myScheduler struct {
	//池的尺寸
	poolSize uint32
	//通道的长度(也即容量)
	channelLen uint
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

//创建调度器
func NewScheduler() Scheduler {
	return &myScheduler{}
}

func (scheduler *myScheduler) Start(channelLen uint,
	poolSize uint32,
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

	if channelLen == 0 {
		return errors.New("The channel max length (capacity) can not be 0!\n")
	}
	scheduler.channelLen = channelLen
	if poolSize == 0 {
		return errors.New("The pool size can not be 0!\n")
	}
	scheduler.poolSize = poolSize
	scheduler.crawlDepth = crawlDepth
	scheduler.chanman = generateChannelManager(scheduler.channelLen)
	if httpClientGenerator == nil {
		return errors.New("The http client generator list is invalid!")
	}
	dlPool, err := generatePageDownloaderPool(scheduler.poolSize, httpClientGenerator)
	if err != nil {
		errMsg := fmt.Sprintf("Occur error when get page downloader pool:%s\n", err)
		return errors.New(errMsg)
	}
	scheduler.dlPool = dlPool
	analyzerPool, err := generateAnalyzerPool(scheduler.poolSize)
	if err != nil {
		errMsg := fmt.Sprintf("Occur error when get analyzer pool:%s\n", err)
		return errors.New(errMsg)
	}
	scheduler.analyzerPool = analyzerPool
	return 
}
