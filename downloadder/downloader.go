package downloadder

import (
	"summerWebCrawler/base"
	"net/http"
	"summerWebCrawler/middleware"
)

//网页下载器的接口类型
type PageDownloader interface {
	//获得ID
	Id() uint32
	//根据请求下载网页并返回响应
	Download(req base.Request) (*base.Response, error)
}

type myPageDownloader struct {
	//http客户端
	httpClient http.Client
	//Id
	id uint32
}

var (
	downloaderIdGenertor middleware.IdGenertor = middleware.NewIdGenertor()
)

//生成Id
func genDownloaderId() uint32 {
	return downloaderIdGenertor.GetUint32()
}

//创建网页下载器
func NewPageDownloader(client *http.Client) PageDownloader {
	id := genDownloaderId()
	if client == nil {
		client = &http.Client{}
	}
	return &myPageDownloader{
		id:         id,
		httpClient: *client,
	}
}

func (dl *myPageDownloader) Id() uint32 {
	return dl.id
}

func (dl *myPageDownloader) Download(req base.Request) (*base.Response, error) {
	httpReq := req.HttpReq()
	httpResp, err := dl.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	return base.NewResponse(httpResp, req.Depth()), nil
}
