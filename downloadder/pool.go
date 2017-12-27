package downloadder

//网页下载池的接口类型

type PageDownloaderPool interface {
	//从池中取出一个网页下载器
	Take() (PageDownloader, error)
	//把一个网页下载器归还给池
	Return(dl PageDownloader) error
	//获取池的总容量
	Total() uint32
	//获得正在被使用的网页下载器的数据
	Used() uint32
}
