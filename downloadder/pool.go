package downloadder

import (
	"summerWebCrawler/middleware"
	"reflect"
	"fmt"
	"errors"
)

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

//网页下载器池的实现类型
type myDownloaderPool struct {
	//实体池
	pool middleware.Pool
	//池内实体的类型
	etype reflect.Type
}

//生成网页下载器的函数类型
type GenPageDownlaoder func() PageDownloader

//创建网页下载器池
func NewPageDownloaderPool(total uint32, gen GenPageDownlaoder) (PageDownloaderPool, error) {
	//得出pagedownloaderPool支持的实体类型
	eType := reflect.TypeOf(gen())
	genEntity := func() middleware.Entity {
		return gen()
	}
	//初始化网页下载器池
	pool, err := middleware.NewPool(total, eType, genEntity)
	if err != nil {
		return nil, err
	}
	dlPool := &myDownloaderPool{
		pool:  pool,
		etype: eType,
	}
	return dlPool, nil
}

func (dlPool *myDownloaderPool) Take() (PageDownloader, error) {
	//从网页下载器池取出一个实体
	entity, err := dlPool.pool.Take()
	if err != nil {
		return nil, err
	}
	//转换实体格式(存入的实体本来就是pageDownloader类型,只是实现了ID()方法.也是Entity类型)
	//类型断言,转换entity为PageDownloader
	dl, ok := entity.(PageDownloader)
	if !ok {
		errMsg := fmt.Sprintf("The type of entity is not %s\n", dlPool.etype)
		panic(errors.New(errMsg))
	}
	return dl, err
}

func (dlPool *myDownloaderPool) Return(dl PageDownloader) error {
	return dlPool.pool.Return(dl)
}

func (dlPool *myDownloaderPool) Total() uint32 {
	return dlPool.pool.Total()
}

func (dlPool *myDownloaderPool) Used() uint32 {
	return dlPool.pool.Used()
}
