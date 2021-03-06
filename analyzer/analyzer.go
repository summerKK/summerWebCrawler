package analyzer

import (
	"summerWebCrawler/base"
	"net/http"
	"summerWebCrawler/middleware"
	"errors"
	"net/url"
	"summerWebCrawler/logging"
	"fmt"
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

//分析器的实现类型
type myAnalyzer struct {
	//ID
	id uint32
}

var (
	//ID生成器
	analyzerIdGenerator middleware.IdGenertor = middleware.NewIdGenertor()
	//日志记录器
	logger logging.Logger = base.NewLogger()
)

//生成解析器的id
func genAnalyzerID() uint32 {
	return analyzerIdGenerator.GetUint32()
}

//创建分析器
func NewAnalyzer() Analyzer {
	return &myAnalyzer{
		id: genAnalyzerID(),
	}
}

//添加请求值或条目值到列表
func appendDataList(dataList []base.Data, data base.Data, respDepth uint32) []base.Data {
	if data == nil {
		return dataList
	}
	//断言当前data是否是(*base.Request).
	//*base.Request实现了 Vaildate() 方法
	req, ok := data.(*base.Request)
	//如果不是*base.request类型直接加入dataList
	if !ok {
		return append(dataList, data)
	}
	newDepth := respDepth + 1
	//新的请求
	if req.Depth() != newDepth {
		//创建新的请求
		req = base.NewRequest(req.HttpReq(), newDepth)
	}
	return append(dataList, req)

}

//添加错误值到列表
func appendErrorList(errorList []error, err error) []error {
	if err == nil {
		return errorList
	}
	return append(errorList, err)
}

func (analyzer *myAnalyzer) Id() uint32 {
	return analyzer.id
}

func (analyzer *myAnalyzer) Analyze(respParsers []ParseResponse, resp base.Response) (dataList []base.Data, errorList []error) {
	//解析函数不能为nil
	if respParsers == nil {
		err := errors.New("The response parser is invalid")
		return nil, []error{err}
	}
	//获取响应结果
	httpResp := resp.HttpResp()
	if httpResp == nil {
		err := errors.New("The http response is invalid!")
		return nil, []error{err}
	}
	//获取响应的url
	var reqUrl *url.URL = httpResp.Request.URL
	logger.Infof("Parse the response (reqUrl=%s)...\n", reqUrl)
	//获取爬取深度
	respDepth := resp.Depth()
	//respParsers是一个slice[],里面放的是解析函数
	for i, respParser := range respParsers {
		if respParser == nil {
			err := errors.New(fmt.Sprintf("The document parser [%d] is invalid!", i))
			errorList = append(errorList, err)
			continue
		}

		//通过解析函数解析出想要的数据
		pDataList, pErrorList := respParser(httpResp, respDepth)

		if pDataList != nil {
			//把解析的数据加入到dataList列表
			for _, pData := range pDataList {
				//appendDataList()会根据数据类型进行创建新的请求或者直接加入到DataList列表
				dataList = appendDataList(dataList, pData, respDepth)
			}
		}

		if pErrorList != nil {
			for _, pError := range pErrorList {
				errorList = appendErrorList(errorList, pError)
			}
		}
	}
	return dataList, errorList
}
