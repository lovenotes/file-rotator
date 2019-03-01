package filerotator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileRotator struct {
	_file *os.File

	_debug *log.Logger
	_info  *log.Logger
	_warn  *log.Logger
	_err   *log.Logger
	_raw   *log.Logger

	_level int

	_rotate *rotate

	sync.RWMutex
}

type rotate struct {
	_size int64

	_interval time.Duration
	_expired  time.Duration
}

// interval: 支持10分钟/半小时/小时/天
// expired: 日志过期时间, 每小时检查一次
func NewFileRotator(path string, level int, size int64, interval time.Duration, expired time.Duration) *FileRotator {
	if interval != INTERVAL_TYPE_TEN_MINUTE && interval != INTERVAL_TYPE_HALF_HOUR &&
		interval != INTERVAL_TYPE_HOUR && interval != INTERVAL_TYPE_DAY {
		_std_error.Fatalf("new file rotator (%v) err (interval illegal).\n", interval)

		return nil
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		_std_error.Fatalf("new file rotator (%s) err (%v).", path, err)
	}

	fileRotator := &FileRotator{
		_file: file,

		_level: level,

		_rotate: &rotate{
			_size: size,

			_interval: interval,
			_expired:  expired,
		},
	}

	fileRotator.setLogLevel(file, level)

	go fileRotator.monitorFile()

	return fileRotator
}

// 设置Logger级别
func (this *FileRotator) setLogLevel(file *os.File, level int) {
	switch {
	case level >= LOG_LEVEL_DEBUG:
		this._debug = log.New(file, "\033[0;36mDEBUG:\033[0m ", log.LstdFlags|log.Lshortfile)
		fallthrough
	case level >= LOG_LEVEL_INFO:
		this._info = log.New(file, "INFO : ", log.LstdFlags|log.Lshortfile)
		fallthrough
	case level >= LOG_LEVEL_WARN:
		this._warn = log.New(file, "\033[0;35mWARN :\033[0m ", log.LstdFlags|log.Lshortfile)
		fallthrough
	case level >= LOG_LEVEL_ERROR:
		this._err = log.New(file, "\033[0;31mERROR:\033[0m ", log.LstdFlags|log.Lshortfile)
	}

	this._raw = log.New(file, "", 0)

	switch {
	case level < LOG_LEVEL_ERROR:
		this._err = nil
		fallthrough
	case level < LOG_LEVEL_WARN:
		this._warn = nil
		fallthrough
	case level < LOG_LEVEL_INFO:
		this._info = nil
		fallthrough
	case level < LOG_LEVEL_DEBUG:
		this._debug = nil
	}
}

// 获取TLog文件大小
func (this *FileRotator) getFileSize() int64 {
	this.RLock()
	defer this.RUnlock()

	fi, err := this._file.Stat()

	if err != nil {
		_std_warn.Printf("get file size err (stat %v).\n", err)

		return 0
	}

	return fi.Size()
}

// 截断并重命名超过Interval的Log文件
func (this *FileRotator) truncFile(filepath, ext string) {
	this.Lock()
	defer this.Unlock()

	err := this._file.Close()

	if err != nil {
		_std_warn.Printf("trunc file err (close %v).\n", err)

		return
	}

	err = os.Rename(filepath, filepath+ext)

	if err != nil {
		_std_warn.Printf("trunc file err (rename %v).\n", err)
	}

	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		_std_warn.Printf("trunc file err (open %v).\n", err)

		return
	}

	// 重置文件写入
	this.setLogLevel(file, this._level)

	this._file = file
}

// 生成Logg文件后缀
func suffix(t time.Time, duration time.Duration) string {
	year, month, day := t.Date()

	// 支持按照天/小时/分钟切割
	if duration%(24*time.Hour) == 0 {
		return "-" + fmt.Sprintf("%04d%02d%02d", year, month, day)
	} else if duration%time.Hour == 0 {
		return "-" + fmt.Sprintf("%04d%02d%02d%02d", year, month, day, t.Hour())
	}

	return "-" + fmt.Sprintf("%04d%02d%02d%02d%02d", year, month, day, t.Hour(), t.Minute())
}

// 截断获得下一次指定时间段的时间
func toNextBound(duration time.Duration) time.Duration {
	if duration == INTERVAL_TYPE_DAY {
		curTime := time.Now()

		curYear := curTime.Year()
		curMonth := curTime.Month()
		curDay := curTime.Day()

		curStartTime := time.Date(curYear, curMonth, curDay, 0, 0, 0, 0, time.Local)

		return curStartTime.Add(duration).Sub(curTime)
	}

	return time.Now().Truncate(duration).Add(duration).Sub(time.Now())
}

// Log监听处理函数
func (this *FileRotator) monitorFile() error {
	interval := time.After(toNextBound(this._rotate._interval))
	expired := time.After(LOG_CHECK_EXPIRED)

	// 按照文件大小分割文件后缀
	sizeExt := 1

	fn := filepath.Base(this._file.Name())

	fp, err := filepath.Abs(this._file.Name())

	if err != nil {
		_std_error.Fatalf("monitor file err (%v).", err)
	}

	for {
		var size <-chan time.Time

		if toNextBound(this._rotate._interval) != LOG_CHECK_INTERVAL {
			size = time.After(LOG_CHECK_INTERVAL)
		}

		select {
		case t := <-interval:
			// 自定义生成新的Logger文件
			interval = time.After(this._rotate._interval)

			this.truncFile(fp, suffix(t, this._rotate._interval))
			sizeExt = 1

			_std_info.Printf("monitor file info (truncated by interval).\n")
		case <-expired:
			// 删除过期的Logger文件
			expired = time.After(LOG_CHECK_EXPIRED)

			err := filepath.Walk(filepath.Dir(fp),
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return nil
					}

					isLog := strings.Contains(info.Name(), fn)

					if time.Since(info.ModTime()) > this._rotate._expired && isLog && info.IsDir() == false {
						if err := os.Remove(path); err != nil {
							return err
						}

						_std_info.Printf("monitor file (%s) info (remove by expired).\n", path)
					}
					return nil
				})

			if err != nil {
				_std_error.Printf("monitor file err (remove %v).\n", err)
			}
		case t := <-size:
			// 文件大小超过上限
			if this._rotate._size == 0 || this.getFileSize() < this._rotate._size {
				break
			}

			this.truncFile(fp, suffix(t, this._rotate._interval)+"."+strconv.Itoa(sizeExt))

			sizeExt++

			_std_info.Printf("monitor file info (trunc by size).\n")
		}
	}
}

// 输出Debug日志
func (this *FileRotator) Debug(format string, v ...interface{}) {
	this.RLock()
	defer this.RUnlock()

	if this._debug != nil {
		this._debug.Output(3, fmt.Sprintln(fmt.Sprintf(format, v...)))
	}
}

// 输出Info日志
func (this *FileRotator) Info(format string, v ...interface{}) {
	this.RLock()
	defer this.RUnlock()

	if this._info != nil {
		this._info.Output(3, fmt.Sprintln(fmt.Sprintf(format, v...)))
	}
}

// 输出Warn日志
func (this *FileRotator) Warn(format string, v ...interface{}) {
	_std_warn.Output(3, fmt.Sprintln(fmt.Sprintf(format, v...)))

	this.RLock()
	defer this.RUnlock()

	if this._warn != nil {
		this._warn.Output(3, fmt.Sprintln(fmt.Sprintf(format, v...)))
	}
}

// 输出Error日志
func (this *FileRotator) Error(format string, v ...interface{}) {
	_std_error.Output(3, fmt.Sprintln(fmt.Sprintf(format, v...)))

	this.RLock()
	defer this.RUnlock()

	if this._err != nil {
		this._err.Output(3, fmt.Sprintln(fmt.Sprintf(format, v...)))
	}
}

// 输出原始日志
func (this *FileRotator) Raw(format string, v ...interface{}) {
	// _std_raw.Output(3, fmt.Sprintln(fmt.Sprintf(format, v...)))
	this.RLock()
	defer this.RUnlock()

	if this._raw != nil {
		this._raw.Output(3, fmt.Sprintln(fmt.Sprintf(format, v...)))
	}
}
