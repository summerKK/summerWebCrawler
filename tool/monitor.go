package tool

import (
	sched "summerWebCrawler/scheduler"
	"time"
	"errors"
	"fmt"
	"runtime"
)

//日志记录函数的类型
//参数level代表日志级别,级别设定: 0:普通,1:警告,2:错误
type Record func(level byte, content string)

//调度监控器函数
//参数scheduler代表作为监控目标的调度器
//参数intervalNs代表检查间隔时间.单位:纳秒
//参数maxIdleCount 代表最大空闲计数
//擦书autoStop被用来指示该方法是否在调度器空闲一段时间(即持续空闲时间,由intervalNs * maxIdleCount得出)之后自行停止调度器
//参数detailSummary被用来表示是否需要详细的摘要信息
//参数record代表日志记录函数
//当监控结束之后,该方法会向作为唯一返回值的通道发送一个代表了空闲状态检查次数的数值
func Monitoring(scheduler sched.Scheduler, intervalNs time.Duration, maxIdleCount uint, autoStop bool, detailSummary bool, record Record) <-chan uint64 {
	if scheduler == nil {
		panic(errors.New("The sched is invalid!"))
	}
	//防止过小的参数值对爬取流程的影响
	if intervalNs < time.Millisecond {
		intervalNs = time.Millisecond
	}
	if maxIdleCount < 1000 {
		maxIdleCount = 1000
	}
	//监控停止通知器
	stopNotifier := make(chan byte, 1)
	//接收和报告错误
	reportError(scheduler, record, stopNotifier)
	//记录摘要信息
	recordSummary(scheduler, detailSummary, record, stopNotifier)
	//检查计数通道
	checkCountChan := make(chan uint64, 2)
	//检查空闲状态
	checkStatus(scheduler, intervalNs, maxIdleCount, autoStop, checkCountChan, record, stopNotifier)
	return checkCountChan
}

var (
	summaryForMonitoring string = "Monitor - Collected infomation[%d]:\n" +
		" Goroutine number:%d\n " +
		"Scheduler:\n%s" +
		"Escaped time:%s\n"
	//已达到最大空闲计数的消息模板
	msgReachMaxIdleCount string = "The sched has been idle for a period of time" +
		" (about %s)." +
		" Now consider what stop it."
	// 停止调度器的消息模板。
	msgStopScheduler = "Stop sched...%s."
)

func reportError(scheduler sched.Scheduler, record Record, stopNotifier <-chan byte) {
	go func() {
		//等待调度器开启
		waitForSchedulerStart(scheduler)
		for {
			//查看监控停止通知器
			select {
			case <-stopNotifier:
				return
			default:
			}
			errorChan := scheduler.ErrorChan()
			if errorChan == nil {
				return
			}
			err := <-errorChan
			if err != nil {
				errMsg := fmt.Sprintf("Error (received from error channel):%s", err)
				record(2, errMsg)
			}
			time.Sleep(time.Microsecond)
		}
	}()
}

//等待调度器开启
func waitForSchedulerStart(scheduler sched.Scheduler) {
	for !scheduler.Running() {
		time.Sleep(time.Microsecond)
	}
}

//记录摘要信息
func recordSummary(scheduler sched.Scheduler, detailSummary bool, record Record, stopNotifier <-chan byte) {
	go func() {
		//等待调度器开启
		waitForSchedulerStart(scheduler)

		var recordCount uint64 = 1
		startTime := time.Now()
		var prevSchedSummary sched.SchedSummary
		var prevNumGoroutine int

		for {
			//查看监控停止通知器
			select {
			case <-stopNotifier:
				return
			default:
			}
			//获取摘要信息的各组成部分
			currNumGoroutine := runtime.NumGoroutine()
			//当前调度器的简要信息
			currSchedSummary := scheduler.Summary(" ")

			//比对前后两份摘要信息的一致性,只有不一致时才会予以记录
			if currNumGoroutine != prevNumGoroutine ||
				!currSchedSummary.Same(prevSchedSummary) {
				schedulerSummaryStr := func() string {
					if detailSummary {
						return currSchedSummary.Detail()
					} else {
						return currSchedSummary.String()
					}
				}()
				//记录摘要信息
				info := fmt.Sprintf(summaryForMonitoring,
					recordCount,
					currNumGoroutine,
					schedulerSummaryStr,
					time.Since(startTime).String())
				record(0, info)
				prevNumGoroutine = currNumGoroutine
				prevSchedSummary = currSchedSummary
				recordCount++
			}

			time.Sleep(time.Microsecond)
		}
	}()
}

func checkStatus(scheduler sched.Scheduler, intervalNs time.Duration, maxIdleCount uint, autoStop bool, checkCountChan chan<- uint64, record Record, stopNotifier chan<- byte) {
	var checkCount uint64
	go func() {
		defer func() {
			stopNotifier <- 1
			stopNotifier <- 2
		}()

		//等待调度器开启
		waitForSchedulerStart(scheduler)

		var idleCount uint
		var firstIdleTime time.Time
		for {
			//调查调度器的空闲状态
			if scheduler.Idle() {
				idleCount++
				if idleCount == 1 {
					firstIdleTime = time.Now()
				}
				if idleCount >= maxIdleCount {
					msg := fmt.Sprintf(msgReachMaxIdleCount, time.Since(firstIdleTime).String())
					record(0, msg)
					//再次检查调度器的空闲状态,确保它已经可以被停止
					if scheduler.Idle() {
						if autoStop {
							var result string
							if scheduler.Stop() {
								result = "success"
							} else {
								result = "failing"
							}
							msg := fmt.Sprintf(msgStopScheduler, result)
							record(0, msg)
						}
						break
					} else {
						if idleCount > 0 {
							idleCount = 0
						}
					}
				}
			} else {
				if idleCount > 0 {
					idleCount = 0
				}
			}
			checkCount++
			time.Sleep(intervalNs)
		}
	}()
}
