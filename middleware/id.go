package middleware

import (
	"sync"
	"math"
)

//ID生成器接口类型
type IdGenertor interface {
	//获得一个uint32类型的ID
	GetUint32() uint32
}

//ID生成器的实现类型
type cyclicIdGenertor struct {
	//当前ID
	sn uint32
	//前一个ID是否已经为其类型所能表示的最大值
	ended bool
	//互斥锁
	mutex sync.Mutex
}

//创建ID生成器
func NewIdGenertor() IdGenertor {
	return &cyclicIdGenertor{}
}

func (gen *cyclicIdGenertor) GetUint32() uint32 {
	gen.mutex.Lock()
	defer gen.mutex.Unlock()
	if gen.ended {
		defer func() {
			gen.ended = false
		}()
		gen.sn = 0
		return gen.sn
	}

	id := gen.sn
	if id < math.MaxUint32 {
		gen.sn++
	} else {
		gen.ended = true
	}
	return id

}
