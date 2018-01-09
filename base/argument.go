package base

import (
	"errors"
	"fmt"
)

//参数容器接口
type Args interface {
	//自检参数的有效性,并在必要时返回可以说明问题的错误值
	//若结果值为nil,则说明未发现问题.否则就意味着自检未通过
	Check() error
	//获得参数容器的字符串表现形式
	String() string
}

//通道参数的容器
type ChannelArgs struct {
	//请求通道的长度
	reqChanLen uint
	//响应通道的长度
	respChanLen uint
	//条目通道的长度
	itemChanLen uint
	//错误通道的长度
	errorChanLen uint
	//描述
	description string
}

//池基本参数的容器
type PoolBaseArgs struct {
	//网页下载器池的尺寸
	pageDownlaoderPoolSize uint32
	//分析器池的尺寸
	analyzerPoolSize uint32
	//描述
	description string
}

var (
	//通道参数的容器的描述模板
	channelArgsTemplate string = "{reqChanLen:%d, respChanLen:%d, itemChanLen:%d, errorChanLen:%d}"
	//池参数的容器的描述模板
	poolbaseArgsTempate string = "{pageDownloaderPoolSize:%d, analyzerPoolSize:%d}"
)

func NewChannelArgs(reqChanLen, respChanLen, itemChanLen, errorChanLen uint) ChannelArgs {
	return ChannelArgs{
		reqChanLen:   reqChanLen,
		respChanLen:  respChanLen,
		itemChanLen:  itemChanLen,
		errorChanLen: errorChanLen,
	}
}

//获得请求通道的长度
func (args *ChannelArgs) ReqChanLen() uint {
	return args.reqChanLen
}

//获得响应通道的长度
func (args *ChannelArgs) RespChanLen() uint {
	return args.respChanLen
}

//获得条目通道的长度
func (args *ChannelArgs) ItemChanLen() uint {
	return args.itemChanLen
}

//获得错误通道的长度
func (args *ChannelArgs) ErrorChanLen() uint {
	return args.errorChanLen
}

func (args *ChannelArgs) Check() error {
	if args.reqChanLen == 0 {
		return errors.New("The request channel max length (capacity) can not be 0!\n")
	}
	if args.respChanLen == 0 {
		return errors.New("The response channel max length (capacity) can not be 0!\n")
	}
	if args.itemChanLen == 0 {
		return errors.New("The item channel max length (capacity) can not be 0!\n ")
	}
	if args.errorChanLen == 0 {
		return errors.New("The error channel max length (capacity) can not be 0!\n")
	}
	return nil
}

func (args *ChannelArgs) String() string {
	if args.description == "" {
		args.description = fmt.Sprintf(channelArgsTemplate,
			args.reqChanLen,
			args.respChanLen,
			args.itemChanLen,
			args.errorChanLen)
	}
	return args.description
}

func NewPoolBaseArgs(pageDownloaderPoolSize, analyzerPoolSize uint32) PoolBaseArgs {
	return PoolBaseArgs{
		pageDownlaoderPoolSize: pageDownloaderPoolSize,
		analyzerPoolSize:       analyzerPoolSize,
	}
}

//获得网页下载器池的尺寸
func (args *PoolBaseArgs) PageDownloaderPoolSize() uint32 {
	return args.pageDownlaoderPoolSize
}

//获得分析器池的尺寸
func (args *PoolBaseArgs) AnalyzerPoolSize() uint32 {
	return args.analyzerPoolSize
}

func (args *PoolBaseArgs) Check() error {
	if args.pageDownlaoderPoolSize == 0 {
		return errors.New("The pageDownloaderPoolSize can not be 0!\n")
	}
	if args.analyzerPoolSize == 0 {
		return errors.New("The analyzerPoolSize can not be 0!\n")
	}
	return nil
}

func (args *PoolBaseArgs) String() string {
	if args.description == "" {
		args.description = fmt.Sprintf(poolbaseArgsTempate,
			args.pageDownlaoderPoolSize,
			args.analyzerPoolSize)
	}
	return args.description
}
