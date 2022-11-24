package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type LogLevel int

const (
	TRACE LogLevel = 5
	DEBUG LogLevel = 10
	INFO  LogLevel = 20
	WARN  LogLevel = 30
	ERROR LogLevel = 40
)

var (
	trace    *log.Logger
	dbg      *log.Logger
	info     *log.Logger
	warn     *log.Logger
	err      *log.Logger
	minLevel LogLevel = TRACE
)

func Init(file *os.File, level LogLevel) {
	minLevel = level
	flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
	if file != nil {
		trace = log.New(file, "TRACE ", flags)
		dbg = log.New(file, "DEBUG ", flags)
		info = log.New(file, "INFO  ", flags)
		warn = log.New(file, "WARN  ", flags)
		err = log.New(file, "ERROR ", flags)
	}
}

func ParseLevel(value string) (LogLevel, error) {
	switch strings.ToLower(value) {
	case "trace":
		return TRACE, nil
	case "debug":
		return DEBUG, nil
	case "info":
		return INFO, nil
	case "warn", "warning":
		return WARN, nil
	case "err", "error":
		return ERROR, nil
	}
	return 0, fmt.Errorf("%s: invalid log level", value)
}

func ErrorLogger() *log.Logger {
	if err == nil {
		return log.New(io.Discard, "", log.LstdFlags)
	}
	return err
}

func Tracef(message string, args ...interface{}) {
	if trace == nil || minLevel > TRACE {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	trace.Output(2, message) //nolint:errcheck // we can't do anything with what we log
}

func Debugf(message string, args ...interface{}) {
	if dbg == nil || minLevel > DEBUG {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	dbg.Output(2, message) //nolint:errcheck // we can't do anything with what we log
}

func Infof(message string, args ...interface{}) {
	if info == nil || minLevel > INFO {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	info.Output(2, message) //nolint:errcheck // we can't do anything with what we log
}

func Warnf(message string, args ...interface{}) {
	if warn == nil || minLevel > WARN {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	warn.Output(2, message) //nolint:errcheck // we can't do anything with what we log
}

func Errorf(message string, args ...interface{}) {
	if err == nil || minLevel > ERROR {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	err.Output(2, message) //nolint:errcheck // we can't do anything with what we log
}
