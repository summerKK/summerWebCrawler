package downloadder

import "summerWebCrawler/base"

//网页下载器的接口类型
type PageDownloader interface {
	//获得ID
	Id() uint32
	//根据请求下载网页并返回响应
	Download(req base.Request) (*base.Response, error)
}
