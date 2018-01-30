package base

import "net/http"

//数据的接口
type Data interface {
	//数据是否有效
	Valid() bool
}

//请求
type Request struct {
	//http请求的指针位置
	httpReq *http.Request
	//请求深度
	depth uint32
}

//响应
type Response struct {
	httpResp *http.Response
	depth    uint32
}

//条目
type Item map[string]interface{}

//创建新的请求
func NewRequest(httpReq *http.Request, depth uint32) *Request {
	return &Request{httpReq: httpReq, depth: depth}
}

//创建新的响应
func NewResponse(httpResp *http.Response, depth uint32) *Response {
	return &Response{httpResp: httpResp, depth: depth}
}

//获取请求
func (req *Request) HttpReq() *http.Request {
	return req.httpReq
}

//获取深度
func (req *Request) Depth() uint32 {
	return req.depth
}

//数据是否有效
func (req *Request) Valid() bool {
	return req.httpReq != nil && req.httpReq.URL != nil
}



//获取http响应
func (resp *Response) HttpResp() *http.Response {
	return resp.httpResp
}

//获取深度值
func (resp *Response) Depth() uint32 {
	return resp.depth
}

//数据是否有效
func (resp *Response) Valid() bool {
	return resp.httpResp != nil && resp.httpResp.Body != nil
}

//数据是否有效
func (item Item) Valid() bool {
	return item != nil
}
