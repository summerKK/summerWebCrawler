package scheduler

import (
	"summerWebCrawler/analyzer"
	"summerWebCrawler/middleware"
	"summerWebCrawler/downloadder"
)

func generateAnalyzerPool(poolSize uint32) (analyzer.AnalyzerPool, error) {
	analyzer, err := analyzer.NewAnalyzerPool(
		poolSize,
		func() analyzer.Analyzer {
			return analyzer.NewAnalyzer()
		},
	)
	if err != nil {
		return nil, err
	}
	return analyzer, nil
}

func generatePageDownloaderPool(poolSize uint32, client GenHttpClient) (downloadder.PageDownloaderPool, error) {
	downloader, err := downloadder.NewPageDownloaderPool(
		poolSize,
		func() downloadder.PageDownloader {
			return downloadder.NewPageDownloader(client())
		},
	)
	if err != nil {
		return nil, err
	}
	return downloader, nil

}

func generateChannelManager(channelLen uint) middleware.ChannelManager {
	return middleware.NewChannelManager(channelLen)
}
