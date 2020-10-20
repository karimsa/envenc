package logger

import (
	"log"
	"os"
)

type LogLevel int

const (
	LevelNone LogLevel = iota
	LevelInfo
	LevelDebug
)

type Logger struct {
	level       LogLevel
	debugLogger *log.Logger
	infoLogger  *log.Logger
}

func New(level LogLevel) Logger {
	return Logger{
		level:       level,
		debugLogger: log.New(os.Stderr, "debug: ", 0),
		infoLogger:  log.New(os.Stdout, "info: ", 0),
	}
}

func (l Logger) Infof(msg string, vals ...interface{}) {
	if l.level >= LevelInfo {
		l.infoLogger.Printf(msg, vals...)
	}
}

func (l Logger) Debugf(msg string, vals ...interface{}) {
	if l.level >= LevelDebug {
		l.debugLogger.Printf(msg, vals...)
	}
}
