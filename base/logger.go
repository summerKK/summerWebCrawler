package base

import "summerWebCrawler/logging"

//创建日志记录器
func NewLogger() logging.Logger {
	return logging.NewSimpleLogger()
}
