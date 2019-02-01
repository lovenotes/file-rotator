## 文件截断库

### Demo: 
	package main
	
	import (
		"time"
	
		"github.com/lovernote/file-rotator"
	)
	// expired 日志过期时间, 每小时检查一次
	// interval 支持10分钟/半小时/小时/天
	func main() {
		fileRotator := filerotator.NewFileRotator("log.log", filerotator.LOG_LEVEL_INFO,
			0, filerotator.INTERVAL_TYPE_HALF_HOUR, time.Hour * 24 * 7)
	
		for {
			time.Sleep(time.Second)
	
			fileRotator.Raw("hello world, cur time: %v.", time.Now().Unix())
		}
	}