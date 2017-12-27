package itempipeline

import "summerWebCrawler/base"

//条目处理管道的接口类型
type ItemPipeline interface {
	//发送条目
	Send(item base.Item) []error
	//FailFast 方法会返回一个布尔值.该值标识当前的条目处理管道是否是快速失败的
	//快速失败:只要对某个条目的处理流程在某一个步骤上出错
	//那么条目处理管道就会忽略掉后续的所有处理步骤并报告错误
	FailFast() bool
	//设置是否快速失败
	SetFailFast(failFast bool)
	//获得已发送丶已接受和已处理的条目的计数值
	//更确切的说,作为结果只的切片总会有3个元素值.这三个值会分别代表前述的3个计数
	Count() []uint64
	//获取正在被处理的条目的数量
	ProcessingNumber() uint64
	//获取摘要信息
	Summary() string
}


