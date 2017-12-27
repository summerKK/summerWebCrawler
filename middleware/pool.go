package middleware

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
