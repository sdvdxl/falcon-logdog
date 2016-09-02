package log

import (
	"fmt"
	"github.com/sdvdxl/falcon-logdog/config"
	"github.com/sdvdxl/log4go"
	"strings"
)

var (
	logger log4go.Logger
)

func init() {
	//初始化日志
	logger = log4go.NewConsoleLogger(log4go.DEBUG)

	logger.Debug("log level %s", config.Cfg.LogLevel)
	logLevel := log4go.TRACE

	switch config.Cfg.LogLevel {
	case log4go.FINEST.String():
		fallthrough
	case "FINEST":
		logLevel = log4go.FINEST
	case log4go.FINE.String():
		fallthrough
	case "FINE":
		logLevel = log4go.FINE
	case log4go.DEBUG.String():
		fallthrough
	case "DEBUG":
		logLevel = log4go.DEBUG
	case log4go.TRACE.String():
		fallthrough
	case "TRACE":
		logLevel = log4go.TRACE
	case log4go.INFO.String():
		fallthrough
	case "INFO":
		logLevel = log4go.INFO
	case log4go.WARNING.String():
		fallthrough
	case "WARN":
		logLevel = log4go.WARNING
	case log4go.ERROR.String():
		fallthrough
	case "ERROR":
		logLevel = log4go.ERROR
	case log4go.CRITICAL.String():
		fallthrough
	case "FATAL":
		logLevel = log4go.CRITICAL
	default:
		logLevel = log4go.INFO
	}

	logger = log4go.NewConsoleLogger(logLevel)
	logger.Info("log inited , level： %v", logLevel)

}

func Info(arg interface{}, args ...interface{}) {
	logger.Logf(log4go.INFO, fmt.Sprint(arg)+strings.Repeat(" %v", len(args)), args...)
}

func Infof(format string, args ...interface{}) {
	logger.Logf(log4go.INFO, format, args...)
}

func Error(arg interface{}, args ...interface{}) {
	logger.Logf(log4go.ERROR, fmt.Sprint(arg)+strings.Repeat(" %v", len(args)), args...)
}

func Errorf(format string, args ...interface{}) {
	logger.Logf(log4go.ERROR, format, args...)
}

func Warn(arg interface{}, args ...interface{}) {
	logger.Logf(log4go.WARNING, fmt.Sprint(arg)+strings.Repeat(" %v", len(args)), args...)
}

func Warnf(format string, args ...interface{}) {
	logger.Logf(log4go.WARNING, format, args...)
}

func Fatal(arg interface{}, args ...interface{}) {
	logger.Logf(log4go.CRITICAL, fmt.Sprint(arg)+strings.Repeat(" %v", len(args)), args...)
}

func Fatalf(format string, args ...interface{}) {
	logger.Logf(log4go.CRITICAL, format, args...)
}

func Debug(arg interface{}, args ...interface{}) {
	logger.Logf(log4go.DEBUG, fmt.Sprint(arg)+strings.Repeat(" %v", len(args)), args...)
}

func Debugf(format string, args ...interface{}) {
	logger.Logf(log4go.DEBUG, format, args...)
}

func Close() {
	logger.Close()
}
