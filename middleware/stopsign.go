package middleware

import (
	"sync"
	"fmt"
)

//停止信号的接口类型
type StopSign interface {
	//置位停止信号,相当于发出停止信号
	//如果先前已发出过停止信号,那么该方法会返回false
	Sign() bool
	//判断停止信号是否已被发出
	Signed() bool
	//重置停止信号.相当于收回停止信号.并清除所有的停止信号处理记录
	Reset()
	//处理停止信号
	//参数code应该代表停止信号处理方的代号.该代号会出现在停止信号的处理记录中
	Deal(code string)
	//获取某一个停止信号处理方的处理计数,该处理计数会从相应的停止信号处理记录中获得
	DealCount(code string) uint32
	//获取停止信号被处理的总计数
	DealTotal() uint32
	//获取摘要信息.其中应该包含所有的停止信号处理记录
	Summary() string
}

type myStopSign struct {
	//表示信号是否已经发出的标志位(true表示停止)
	signed bool
	//处理计数的字典
	dealCountMap map[string]uint32
	//读写锁
	rwmutex sync.RWMutex
}

//创建停止信号
func NewStopSign() StopSign {
	ss := &myStopSign{
		dealCountMap: make(map[string]uint32),
	}
	return ss
}

func (ss *myStopSign) Sign() bool {
	ss.rwmutex.Lock()
	defer ss.rwmutex.Unlock()
	if ss.signed {
		return false
	}
	ss.signed = true
	return true
}

func (ss *myStopSign) Signed() bool {
	return ss.signed
}

//处理停止信号
//参数code应该代表停止信号处理方的代号.该代号会出现在停止信号的处理记录中
func (ss *myStopSign) Deal(code string) {
	ss.rwmutex.Lock()
	defer ss.rwmutex.Unlock()
	//停止信号还未被发出,忽略后续操作.保证计数的准确性
	if !ss.signed {
		return
	}
	if _, ok := ss.dealCountMap[code]; !ok {
		ss.dealCountMap[code] = 1
	} else {
		ss.dealCountMap[code] += 1
	}
}

//重置停止信号
func (ss *myStopSign) Reset() {
	ss.rwmutex.Lock()
	defer ss.rwmutex.Unlock()
	ss.signed = false
	ss.dealCountMap = make(map[string]uint32)
}

func (ss *myStopSign) DealCount(code string) uint32 {
	ss.rwmutex.Lock()
	defer ss.rwmutex.Unlock()
	return ss.dealCountMap[code]
}

func (ss *myStopSign) DealTotal() uint32 {
	ss.rwmutex.Lock()
	defer ss.rwmutex.Unlock()
	var total uint32
	for _, v := range ss.dealCountMap {
		total += v
	}
	return total
}

func (ss *myStopSign) Summary() string {
	if ss.signed {
		return fmt.Sprintf("signed:true,dealCount:%v", ss.dealCountMap)
	}
	return "signed:false"
}
