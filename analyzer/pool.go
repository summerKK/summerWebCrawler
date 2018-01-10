package analyzer

import (
	"summerWebCrawler/middleware"
	"reflect"
	"fmt"
	"errors"
)

//分析池的接口类型
type AnalyzerPool interface {
	//从池中取出一个分析器
	Take() (Analyzer, error)
	//把一个分析器归还给池
	Return(analyzer Analyzer) error
	//获得池的总容量
	Total() uint32
	//获得正在使用的分析器的数量
	Used() uint32
}

//分析器的实现
type myAnalyzerPool struct {
	//实体池
	pool middleware.Pool
	//池内实体的类型
	eType reflect.Type
}

type GenAnalyzer func() Analyzer

//创建分析器池
func NewAnalyzerPool(total uint32, gen GenAnalyzer) (AnalyzerPool, error) {
	//获取生成实体类型
	eType := reflect.TypeOf(gen())
	//池中的实体
	genEntity := func() middleware.Entity {
		return gen()
	}
	pool, err := middleware.NewPool(total, eType, genEntity)
	if err != nil {
		return nil, err
	}
	maPool := &myAnalyzerPool{
		pool:  pool,
		eType: eType,
	}
	return maPool, nil

}

func (maPool *myAnalyzerPool) Take() (Analyzer, error) {
	entity, err := maPool.pool.Take()
	if err != nil {
		return nil, err
	}
	//判断实体是否是Analyzer类型
	v, ok := entity.(Analyzer)
	if !ok {
		errMsg := fmt.Sprintf("The type of entity is not %s\n", maPool.eType)
		//这里返回错误的实体直接panic程序.避免后续大问题
		panic(errors.New(errMsg))
	}
	return v, nil

}

func (maPool *myAnalyzerPool) Return(ma Analyzer) error {
	return maPool.pool.Return(ma)
}

func (maPool *myAnalyzerPool) Total() uint32 {
	return maPool.pool.Total()
}

func (maPool *myAnalyzerPool) Used() uint32 {
	return maPool.pool.Used()
}
