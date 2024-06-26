package log_plus

import (
	"log"
	"os"
	"sync"
)

var Grade uint8 = 0
var LogFile string

var (
	logger     *log.Logger
	loggerOnce sync.Once
)

func getLogger() *log.Logger {
	loggerOnce.Do(func() {
		var fd *os.File
		if LogFile != "" {
			if _fd, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err != nil {
				panic(err)
			} else {
				fd = _fd
			}
		} else {
			fd = os.Stdout
		}
		logger = log.New(fd, "", log.Ldate|log.Ltime)
	})
	return logger
}

const (
	DEBUG_MONITOR   = 1 << 0
	SUBMIT_RETURN   = 1 << 1
	DEBUG_LEADER    = 1 << 2
	DEBUG_WORKER    = 1 << 3
	DEBUG_TIME_HEAP = 1 << 4
	DEBUG_OTHER     = 1 << 5
)

func Printf(grade uint8, format string, v ...any) {
	if (grade & Grade) > 0 {
		getLogger().Printf(format, v...)
	}
}

func Print(grade uint8, v ...any) {
	if (grade & Grade) > 0 {
		getLogger().Print(v...)
	}
}

func Println(grade uint8, v ...any) {
	if (grade & Grade) > 0 {
		getLogger().Println(v...)
	}
}
