package logging

import (
	"fmt"
	"log"

	"github.com/hashicorp/logutils"
)

type LogLevel string

const (
	Trace = "TRACE"
	Debug = "DEBUG"
	Info  = "INFO"
	Warn  = "WARN"
	Error = "ERROR"
)

var Levels = []logutils.LogLevel{Trace, Debug, Info, Warn, Error}

func ByName(n string) logutils.LogLevel {
	for _, l := range Levels {
		if n == string(l) {
			return l
		}
	}
	return Trace
}

type Logger struct {
	simple  bool
	client  string
	session string
	format  string
}

func New(sessid, client string) *Logger {
	return &Logger{
		simple:  false,
		client:  client,
		session: sessid,
		format:  "[%s] [client=%s] [sid=%s] %s",
	}
}

func Simple() *Logger {
	return &Logger{
		simple: true,
		format: "[%s] %s",
	}
}

func (l *Logger) logimpl(level LogLevel, format string, data ...interface{}) {
	if l.simple {
		log.Printf(l.format, level, fmt.Sprintf(format, data...))
	} else {
		log.Printf(l.format, level, l.client, l.session, fmt.Sprintf(format, data...))
	}
}

func (l *Logger) Log(level LogLevel, format string, data ...interface{}) {
	l.logimpl(level, format, data...)
}

func (l *Logger) Tracef(format string, data ...interface{}) {
	l.Log(Trace, format, data...)
}

func (l *Logger) Debugf(format string, data ...interface{}) {
	l.Log(Debug, format, data...)
}

func (l *Logger) Infof(format string, data ...interface{}) {
	l.Log(Info, format, data...)
}

func (l *Logger) Warnf(format string, data ...interface{}) {
	l.Log(Warn, format, data...)
}

func (l *Logger) Errorf(format string, data ...interface{}) {
	l.Log(Error, format, data...)
}
