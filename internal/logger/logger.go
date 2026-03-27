package logger

import (
	"log"         // Default(), log.Logger
	"sync/atomic" // atomic.Pointer
)

var loggerSingleton atomic.Pointer[log.Logger]

func init() { // https://go.dev/doc/effective_go#init
	loggerSingleton.Store(log.Default())
}

func Get() *log.Logger {
	return loggerSingleton.Load()
}

func Set(instance *log.Logger) {
	if instance == nil {
		loggerSingleton.Store(log.Default())
	} else {
		loggerSingleton.Store(instance)
	}
}
