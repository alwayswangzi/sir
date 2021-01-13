package sir

import (
	"log"
	"runtime"
)

func logs(level string, i interface{}, skip int) {
	_, file, line, ok := runtime.Caller(skip)
	if ok {
		log.Printf("[%s] %s:%d %v", level, file, line, i)
	} else {
		log.Printf("[%s] %v", level, i)
	}
}

func LogError(err error, skip ...int) {
	if len(skip) == 0 {
		logs("Error", err, 2)
	} else {
		logs("Error", err, skip[0])
	}
}

func LogInfo(i interface{}) {
	logs("Info", i, 2)
}
