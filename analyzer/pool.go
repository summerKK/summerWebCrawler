package analyzer

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
