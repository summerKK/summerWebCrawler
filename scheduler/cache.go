package scheduler

import (
	"summerWebCrawler/base"
	"sync"
	"fmt"
)

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

type reqCacheBySlice struct {
	cache []*base.Request
	mutex sync.Mutex
	//代表请求状态0代表初始化,1代表关闭
	status byte
}

var (
	summaryTemplate = "status:%s," + "length:%d," + "capacity:%d"
	//状态字典
	statusMap = map[byte]string{
		0: "running",
		1: "closed",
	}
)

//创建请求缓存
func NewRequestCache() requestCache {
	rc := &reqCacheBySlice{
		cache: make([]*base.Request, 0),
	}
	return rc
}

func (reqcache *reqCacheBySlice) put(req *base.Request) bool {
	if req == nil {
		return false
	}
	if reqcache.status == 1 {
		return false
	}
	reqcache.mutex.Lock()
	defer reqcache.mutex.Unlock()
	reqcache.cache = append(reqcache.cache, req)
	return true
}

func (reqcache *reqCacheBySlice) get() *base.Request {
	if reqcache.length() == 0 {
		return nil
	}
	if reqcache.status == 1 {
		return nil
	}
	reqcache.mutex.Lock()
	defer reqcache.mutex.Unlock()
	req := reqcache.cache[0]
	reqcache.cache = reqcache.cache[1:]
	return req
}

func (reqcache *reqCacheBySlice) capacity() int {
	return cap(reqcache.cache)
}

func (reqcache *reqCacheBySlice) length() int {
	return len(reqcache.cache)
}

func (reqcache *reqCacheBySlice) close() {
	if reqcache.status == 1 {
		return
	}
	reqcache.status = 1
}

func (reqcache *reqCacheBySlice) summary() string {
	summary := fmt.Sprintf(summaryTemplate,
		statusMap[reqcache.status],
		reqcache.length(),
		reqcache.capacity())
	return summary
}
