package main

import (
	"summerWebCrawler/base"
	"net/http"
	"summerWebCrawler/logging"
	"errors"
	"fmt"
	"net/url"
	"io"
	"github.com/PuerkitoBio/goquery"
	"strings"
	"summerWebCrawler/analyzer"
	"time"
	"summerWebCrawler/itempipeline"
	"summerWebCrawler/scheduler"
	"summerWebCrawler/tool"
)

var (
	//日志记录器
	logger logging.Logger = logging.NewSimpleLogger()
)

func main() {

	//创建调度器
	scheduler := scheduler.NewScheduler()
	//准备监控参数
	intervalNs := 10 * time.Millisecond
	maxIdleCount := uint(1000)
	//开始监控
	checkCountChan := tool.Monitoring(
		scheduler,
		intervalNs,
		maxIdleCount,
		true,
		false,
		record)

	//准备启动参数
	channelArgs := base.NewChannelArgs(10, 10, 10, 10)
	poolBaseArgs := base.NewPoolBaseArgs(3, 3)
	crawlDepth := uint32(1)
	httpClientGenerator := genHttpClient
	respParses := genResponseParsers()
	itemProcessors := genItemProcessors()
	startUrl := "http://www.sogou.com"
	firstHttpReq, err := http.NewRequest("GET", startUrl, nil)
	if err != nil {
		logger.Errorln(err)
		return
	}

	//开启调度器
	scheduler.Start(
		channelArgs,
		poolBaseArgs,
		crawlDepth,
		httpClientGenerator,
		respParses,
		itemProcessors,
		firstHttpReq)

	//等待监控结束
	<-checkCountChan
}

//生成http客户端
func genHttpClient() *http.Client {
	return &http.Client{}
}

//获得响应解析函数的序列
func genResponseParsers() []analyzer.ParseResponse {
	parsers := []analyzer.ParseResponse{
		parseForATag,
	}
	return parsers
}

//获得条目处理器的序列
func genItemProcessors() []itempipeline.ProcessItem {
	itemProcessors := []itempipeline.ProcessItem{
		processItem,
	}
	return itemProcessors
}

//条目处理器
func processItem(item base.Item) (result base.Item, err error) {
	if item == nil {
		return nil, errors.New("Invalid item!")
	}
	//生成结果
	result = make(map[string]interface{})
	for k, v := range item {
		result[k] = v
	}
	if _, ok := result["number"]; !ok {
		result["number"] = len(result)
	}
	time.Sleep(10 * time.Millisecond)
	return result, nil
}

//响应解析函数,只解析"A"标签
func parseForATag(httpResp *http.Response, respDepth uint32) ([]base.Data, []error) {
	//TODO支持更多的http响应状态
	if httpResp.StatusCode != 200 {
		err := errors.New(fmt.Sprintf("unsupported status code %d. (httpResponseCode=%d)", httpResp.StatusCode))
		return nil, []error{err}
	}

	var reqUrl *url.URL = httpResp.Request.URL
	var httpRespBody io.ReadCloser = httpResp.Body
	defer func() {
		if httpRespBody != nil {
			httpRespBody.Close()
		}
	}()
	dataList := make([]base.Data, 0)
	errs := make([]error, 0)

	//开始解析
	doc, err := goquery.NewDocumentFromReader(httpRespBody)
	if err != nil {
		errs = append(errs, err)
		return dataList, errs
	}

	//查找"A"标签并提取链接地址
	doc.Find("a").Each(func(index int, selection *goquery.Selection) {
		href, exists := selection.Attr("href")
		//前期过滤
		if !exists || href == "" || href == "#" || href == "/" {
			return
		}
		href = strings.TrimSpace(href)
		lowerHref := strings.ToLower(href)
		//暂不支持对JavaScript代码的解析
		if href != "" && !strings.HasPrefix(lowerHref, "javascript") {
			aUrl, err := url.Parse(href)
			if err != nil {
				errs = append(errs, err)
				return
			}
			if !aUrl.IsAbs() {
				aUrl = reqUrl.ResolveReference(aUrl)
			}
			httpReq, err := http.NewRequest("GET", aUrl.String(), nil)
			if err != nil {
				errs = append(errs, err)
			} else {
				req := base.NewRequest(httpReq, respDepth)
				dataList = append(dataList, req)
			}
		}
		text := strings.TrimSpace(selection.Text())
		if text != "" {
			imap := make(map[string]interface{})
			imap["a.text"] = text
			imap["parent_url"] = reqUrl
			item := base.Item(imap)
			dataList = append(dataList, &item)
		}
	})
	return dataList, errs
}

func record(level byte, content string) {
	if content == ""{
		return
	}
	switch level {
	case 0:
		logger.Infoln(content)
	case 1:
		logger.Warnln(content)
	case 2:
		logger.Infoln(content)
	}
}
