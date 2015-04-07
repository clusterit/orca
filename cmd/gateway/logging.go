package main

import "github.com/clusterit/orca/logging"

var (
	logger = logging.Simple()
)

func Log(level logging.LogLevel, format string, data ...interface{}) {
	logger.Log(level, format, data...)
}

func (cs *clientSession) log(level logging.LogLevel, format string, data ...interface{}) {
	cs.logger.Log(level, format, data)
}

func (cs *clientSession) tracef(format string, data ...interface{}) {
	cs.log(logging.Trace, format, data...)
}

func (cs *clientSession) debugf(format string, data ...interface{}) {
	cs.log(logging.Debug, format, data...)
}

func (cs *clientSession) infof(format string, data ...interface{}) {
	cs.log(logging.Info, format, data...)
}

func (cs *clientSession) warnf(format string, data ...interface{}) {
	cs.log(logging.Warn, format, data...)
}

func (cs *clientSession) errorf(format string, data ...interface{}) {
	cs.log(logging.Error, format, data...)
}
