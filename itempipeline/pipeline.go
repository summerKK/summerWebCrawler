package itempipeline

import (
	"summerWebCrawler/base"
	"errors"
	"fmt"
	"sync/atomic"
)

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

//条目处理管道的实现类型
type myItemPipeline struct {
	//条目处理器的列表
	itemProcessors []ProcessItem
	//表示处理器是否需要快速失败的标志位
	failFast bool
	//已被发送的条目的数量
	sent uint64
	//已被接受的条目的数量
	accepted uint64
	//已被处理条目的数量
	processed uint64
	//正在被处理的条目的数量
	processingNumer uint64
}

var summaryTemplate = "failFast: %v, processorNumber: %d," +
	" sent: %d, accepted: %d, processed: %d, processingNumber: %d"

//创建条目处理管道
func NewItemPipeline(itemProcessors []ProcessItem) ItemPipeline {
	if itemProcessors == nil {
		//这里如果发现条目处理器列表为空直接panic结束程序
		panic(errors.New(fmt.Sprintln("Invalid item processor list!")))
	}
	innerItemProcessors := make([]ProcessItem, 0)
	for i, ip := range itemProcessors {
		if ip == nil {
			panic(errors.New(fmt.Sprintf("Invalid item processor[%d]!\n", i)))
		}
		innerItemProcessors = append(innerItemProcessors, ip)
	}
	return &myItemPipeline{itemProcessors: itemProcessors}
}

func (maPool *myItemPipeline) Send(item base.Item) []error {
	atomic.AddUint64(&maPool.processingNumer, 1)
	defer atomic.AddUint64(&maPool.processingNumer, ^uint64(0))
	atomic.AddUint64(&maPool.sent, 1)
	errs := make([]error, 0)
	if item == nil {
		errs = append(errs, errors.New("The item is invalid!"))
		return errs
	}
	var currentItem base.Item = item
	atomic.AddUint64(&maPool.accepted, 1)
	for _, itemProcessor := range maPool.itemProcessors {
		processedItem, err := itemProcessor(currentItem)
		if err != nil {
			errs = append(errs, err)
			if maPool.failFast {
				break
			}
		}
		if processedItem != nil {
			currentItem = processedItem
		}
	}
	atomic.AddUint64(&maPool.processed, 1)
	return errs
}

func (maPool *myItemPipeline) FailFast() bool {
	return maPool.failFast
}

func (maPool *myItemPipeline) SetFailFast(failFast bool) {
	maPool.failFast = failFast
}

func (maPool *myItemPipeline) Count() []uint64 {
	counts := make([]uint64, 3)
	counts[0] = atomic.LoadUint64(&maPool.sent)
	counts[1] = atomic.LoadUint64(&maPool.accepted)
	counts[2] = atomic.LoadUint64(&maPool.processed)
	return counts
}

func (maPool *myItemPipeline) ProcessingNumber() uint64 {
	return atomic.LoadUint64(&maPool.processingNumer)
}

func (ip *myItemPipeline) Summary() string {
	counts := ip.Count()
	summary := fmt.Sprintf(summaryTemplate,
		ip.failFast, len(ip.itemProcessors),
		counts[0], counts[1], counts[2], ip.ProcessingNumber())
	return summary
}
