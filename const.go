package filerotator

import (
	"log"
	"os"
	"time"
)

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
)

const (
	LOG_LEVEL_ERROR = 1
	LOG_LEVEL_WARN  = 2
	LOG_LEVEL_INFO  = 3
	LOG_LEVEL_DEBUG = 4
)

const (
	LOG_CHECK_INTERVAL = time.Minute * 2
	LOG_CHECK_EXPIRED  = time.Hour
)

const (
	INTERVAL_TYPE_TEN_MINUTE = time.Minute * 10
	INTERVAL_TYPE_HALF_HOUR  = time.Minute * 30
	INTERVAL_TYPE_HOUR       = time.Hour
	INTERVAL_TYPE_DAY        = time.Hour * 24
)

var (
	_std_error = log.New(os.Stderr, "\033[0;31mERROR:\033[0m ", log.LstdFlags|log.Lshortfile)
	_std_info  = log.New(os.Stderr, "INFO : ", log.LstdFlags|log.Lshortfile)
	_std_warn  = log.New(os.Stderr, "\033[0;35mWARN :\033[0m ", log.LstdFlags|log.Lshortfile)
	_std_raw   = log.New(os.Stderr, "", 0)
)
