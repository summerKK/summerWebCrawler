package scheduler

import (
	"summerWebCrawler/analyzer"
	"summerWebCrawler/itempipeline"
	"net/http"
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
