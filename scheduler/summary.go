package scheduler

type SchedSummary interface {
	//获得摘要信息的一般表示
	String() string
	//获得摘要信息的详细表示
	Detail() string
	//判断是否与另一份摘要信息相同
	Same(other SchedSummary) bool
}

//获取摘要信息
func NewSchedSummary(sched *myScheduler, prefix string) SchedSummary {

}
