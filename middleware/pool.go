package middleware

import (
	"reflect"
	"errors"
	"fmt"
	"sync"
)

//实体池的接口类型
type Pool interface {
	//取出实体
	Take() (Entity, error)
	//归还实体
	Return(entity Entity) error
	//实体池的总量
	Total() uint32
	//实体池中已经被使用的实体的数量
	Used() uint32
}

//实体接口类型
type Entity interface {
	Id() uint32
}

//实体池的实现类型
type myPool struct {
	//池的总容量
	totla uint32
	//池中实体的类型
	etype reflect.Type
	//池中实体的生成函数
	genEntity func() Entity
	//实体容器
	container chan Entity
	//实体Id的容器
	IDContainer map[uint32]bool
	//针对实体ID容器操作的互斥锁
	mutex sync.Mutex
}

//创建实体池
func NewPool(total uint32, entityType reflect.Type, genEntity func() Entity) (Pool, error) {

	//检查池的容量是否合法
	if total == 0 {
		errMsg := fmt.Sprintf("The pool can not be initialized!(total= %d)\n", total)
		return nil, errors.New(errMsg)
	}
	//初始化container字段
	//通常情况下,我们应该在初始化一个池的时候就把它填满.
	//尤其是在实体的创建成本很低火需要提前准备以尽量减少后续操作的响应时间的情况下
	size := int(total)
	container := make(chan Entity, size)
	idContainer := make(map[uint32]bool)
	for i := 0; i < size; i++ {
		newEntity := genEntity()
		//判断实体生成的类型是否合法
		if entityType != reflect.TypeOf(newEntity) {
			errMsg := fmt.Sprintf("The type of result of function genEntity() is not %s!\n", entityType)
			return nil, errors.New(errMsg)
		}
		container <- newEntity
		//记录实体Id的容器(实体放回资源池的时候用来辨别是否是资源池生成的实体)
		idContainer[newEntity.Id()] = true
	}

	pool := &myPool{
		totla:       total,
		etype:       entityType,
		genEntity:   genEntity,
		container:   container,
		IDContainer: idContainer,
	}

	return pool, nil
}

func (pool *myPool) Take() (Entity, error) {
	//从实体容器返回一个实体
	entity, ok := <-pool.container
	if !ok {
		return nil, errors.New("The inner container is invalid!")
	}
	//加锁
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	//这里改变实体状态,用于辨别是否被拿出或在池中
	pool.IDContainer[entity.Id()] = false
	return entity, nil
}

func (pool *myPool) Return(entity Entity) error {
	//判断资源值是否合法
	if entity == nil {
		return errors.New("The returning entity is invalid!")
	}
	//判断类型是否合法
	if pool.etype != reflect.TypeOf(entity) {
		errMsg := fmt.Sprintf("The type of returning is not %s!\n", pool.etype)
		return errors.New(errMsg)
	}

	entityId := entity.Id()
	casResult := pool.compareAndSetForIDContainer(entityId, false, true)
	if casResult == 1 {
		pool.container <- entity
		return nil
	} else if casResult == 0 {
		errMsg := fmt.Sprintf("The entity (id=%d) is already in the pool\n", entityId)
		return errors.New(errMsg)
	} else {
		errMsg := fmt.Sprintf("The entity (id=%d) is illegal!\n")
		return errors.New(errMsg)
	}
}

//比较并设置实体ID容器中与给定实体ID对应的键值对的元素值
//结果值:
// -1:表示键值对不存在
//  0:表示操作失败
//  1:表示操作成功
func (pool *myPool) compareAndSetForIDContainer(entityId uint32, oldValue bool, newValue bool) int8 {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	v, ok := pool.IDContainer[entityId]
	if !ok {
		return -1
	}
	if v != newValue {
		return 0
	}
	pool.IDContainer[entityId] = newValue
	return 1
}

func (pool *myPool) Total() uint32 {
	return pool.totla
}

func (pool *myPool) Used() uint32 {
	return pool.totla - uint32(len(pool.container))
}
