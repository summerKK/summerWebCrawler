package scheduler

import "summerWebCrawler/base"

//请求缓存类型
type requestCache interface {
	//将请求放入请求缓存
	put(req *base.Request) bool
	//从请求缓存中获最早被放入且人在其中的请求
	get() *base.Request
	//获得请求缓存的容量
	capacity() int
	//获得请求缓存的实时长度,即其中的请求的即时数量
	length() int
	//关闭请求缓存
	close()
	//获取请求缓存的摘要信息
	summary() string
}
