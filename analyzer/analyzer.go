package analyzer

import (
	"summerWebCrawler/base"
	"net/http"
)

//分析器的接口类型
type Analyzer interface {
	//获得ID
	Id() uint32
	//根据规定分析响应并返回请求和条目
	Analyze(
		respParser []ParseResponse,
		resp base.Response) ([]base.Data, []error)
}

//被用于解析http响应的函数类型
type ParseResponse func(httpResp *http.Response, respDepth uint32) ([]base.Data, []error)


