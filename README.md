## 文件截断库

### Demo: 
	package main
	
	import (
		"time"
	
		"github.com/lovernote/file-rotator"
	)
	
	func main() {
		fileRotator := filerotator.NewFileRotator("log.log", filerotator.LOG_LEVEL_INFO,
			0, 24*time.Hour, 7*24*time.Hour)
	
		for {
			time.Sleep(time.Second)
	
			fileRotator.Raw("hello world, cur time: %v.", time.Now().Unix())
		}
	}