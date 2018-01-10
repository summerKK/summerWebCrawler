package scheduler

import (
	analy "summerWebCrawler/analyzer"
	middle "summerWebCrawler/middleware"
	download "summerWebCrawler/downloadder"
	pipe "summerWebCrawler/itempipeline"
	"regexp"
	"strings"
	"errors"
	"fmt"
	"summerWebCrawler/base"
)

var regexpForIp = regexp.MustCompile(`((?:(?:25[0-5]|2[0-4]\d|[01]?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d?\d))`)

var regexpForDomains = []*regexp.Regexp{
	// *.xx or *.xxx.xx
	regexp.MustCompile(`\.(com|com\.\w{2})$`),
	regexp.MustCompile(`\.(gov|gov\.\w{2})$`),
	regexp.MustCompile(`\.(net|net\.\w{2})$`),
	regexp.MustCompile(`\.(org|org\.\w{2})$`),
	// *.xx
	regexp.MustCompile(`\.me$`),
	regexp.MustCompile(`\.biz$`),
	regexp.MustCompile(`\.info$`),
	regexp.MustCompile(`\.name$`),
	regexp.MustCompile(`\.mobi$`),
	regexp.MustCompile(`\.so$`),
	regexp.MustCompile(`\.asia$`),
	regexp.MustCompile(`\.tel$`),
	regexp.MustCompile(`\.tv$`),
	regexp.MustCompile(`\.cc$`),
	regexp.MustCompile(`\.co$`),
	regexp.MustCompile(`\.\w{2}$`),
}

func generateAnalyzerPool(poolSize uint32) (analy.AnalyzerPool, error) {
	analyzer, err := analy.NewAnalyzerPool(
		poolSize,
		func() analy.Analyzer {
			return analy.NewAnalyzer()
		})
	if err != nil {
		return nil, err
	}
	return analyzer, nil
}

//初始化网页下载器池
func generatePageDownloaderPool(poolSize uint32, client GenHttpClient) (download.PageDownloaderPool, error) {
	downloader, err := download.NewPageDownloaderPool(
		poolSize,
		//download.NewPageDownloader(client())返回一个网页下载器(实体)
		//通过实体初始化网页下载器池
		func() download.PageDownloader {
			return download.NewPageDownloader(client())
		})
	if err != nil {
		return nil, err
	}
	return downloader, nil

}

func generateChannelManager(channelArgs base.ChannelArgs) middle.ChannelManager {
	return middle.NewChannelManager(channelArgs)
}

func generateItemPipeline(items []pipe.ProcessItem) pipe.ItemPipeline {
	return pipe.NewItemPipeline(items)
}

func getPrimaryDomain(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", errors.New("The host is empty!")
	}
	if regexpForIp.MatchString(host) {
		return host, nil
	}
	var suffixIndex int
	for _, re := range regexpForDomains {
		pos := re.FindStringIndex(host)
		if pos != nil {
			suffixIndex = pos[0]
			break
		}
	}
	if suffixIndex > 0 {
		var pdIndex int
		firstPart := host[:suffixIndex]
		index := strings.LastIndex(firstPart, ".")
		if index < 0 {
			pdIndex = 0
		} else {
			pdIndex = index + 1
		}
		return host[pdIndex:], nil
	} else {
		return "", errors.New("Unrecognized host!")
	}
}

// 生成组件实例代号。
func generateCode(prefix string, id uint32) string {
	return fmt.Sprintf("%s-%d", prefix, id)
}

// 解析组件实例代号。
func parseCode(code string) []string {
	result := make([]string, 2)
	var codePrefix string
	var id string
	index := strings.Index(code, "-")
	if index > 0 {
		codePrefix = code[:index]
		id = code[index+1:]
	} else {
		codePrefix = code
	}
	result[0] = codePrefix
	result[1] = id
	return result
}
