package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	l  *log.Logger
	lv int
}

const (
	DEBUG = iota
	INFO
	WARN
	ERROR
	FATAL
)

var Log *Logger

func InitLogger(level int) {
	Log = &Logger{
		l:  log.New(os.Stdout, "", 0),
		lv: level,
	}
}

func (lg *Logger) log(level int, tag, msg string, args ...any) {
	if level < lg.lv {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	lg.l.Printf("%s [%s] %s\n", ts, tag, fmt.Sprintf(msg, args...))
	if level == FATAL {
		os.Exit(1)
	}
}

func (lg *Logger) Info(m string, a ...any)  { lg.log(INFO, "INFO", m, a...) }
func (lg *Logger) Error(m string, a ...any) { lg.log(ERROR, "ERROR", m, a...) }
func (lg *Logger) Fatal(m string, a ...any) { lg.log(FATAL, "FATAL", m, a...) }
