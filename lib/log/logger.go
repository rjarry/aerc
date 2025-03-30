package log

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

type logfilePtr struct {
	f         *os.File
	useStdout bool
}

func newLogfilePtr(f *os.File, isStdout bool) *logfilePtr {
	return &logfilePtr{f: f, useStdout: isStdout}
}

func (l *logfilePtr) Close() error {
	if l.useStdout || l.f == nil {
		return nil
	}
	return l.f.Close()
}

var (
	trace    *log.Logger
	dbg      *log.Logger
	info     *log.Logger
	warn     *log.Logger
	err      *log.Logger
	minLevel LogLevel = TRACE

	// logfile stores a pointer to the log file descriptor
	logfile *logfilePtr
)

func Init(file *os.File, useStdout bool, level LogLevel) error {
	trace = nil
	dbg = nil
	info = nil
	warn = nil
	err = nil

	if logfile != nil {
		e := logfile.Close()
		if e != nil {
			return e
		}
		logfile = nil
	}

	minLevel = level
	flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
	if file != nil {
		logfile = newLogfilePtr(file, useStdout)
		trace = log.New(file, "TRACE ", flags)
		dbg = log.New(file, "DEBUG ", flags)
		info = log.New(file, "INFO  ", flags)
		warn = log.New(file, "WARN  ", flags)
		err = log.New(file, "ERROR ", flags)
	}

	return nil
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

type Logger interface {
	Tracef(string, ...any)
	Debugf(string, ...any)
	Infof(string, ...any)
	Warnf(string, ...any)
	Errorf(string, ...any)
}

type logger struct {
	name      string
	calldepth int
}

func NewLogger(name string, calldepth int) Logger {
	return &logger{name: name, calldepth: calldepth}
}

func (l *logger) format(message string, args ...any) string {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	if l.name != "" {
		message = fmt.Sprintf("[%s] %s", l.name, message)
	}
	return message
}

func (l *logger) Tracef(message string, args ...any) {
	if trace == nil || minLevel > TRACE {
		return
	}
	message = l.format(message, args...)
	trace.Output(l.calldepth, message) //nolint:errcheck // we can't do anything with what we log
}

func (l *logger) Debugf(message string, args ...any) {
	if dbg == nil || minLevel > DEBUG {
		return
	}
	message = l.format(message, args...)
	dbg.Output(l.calldepth, message) //nolint:errcheck // we can't do anything with what we log
}

func (l *logger) Infof(message string, args ...any) {
	if info == nil || minLevel > INFO {
		return
	}
	message = l.format(message, args...)
	info.Output(l.calldepth, message) //nolint:errcheck // we can't do anything with what we log
}

func (l *logger) Warnf(message string, args ...any) {
	if warn == nil || minLevel > WARN {
		return
	}
	message = l.format(message, args...)
	warn.Output(l.calldepth, message) //nolint:errcheck // we can't do anything with what we log
}

func (l *logger) Errorf(message string, args ...any) {
	if err == nil || minLevel > ERROR {
		return
	}
	message = l.format(message, args...)
	err.Output(l.calldepth, message) //nolint:errcheck // we can't do anything with what we log
}

var root = logger{calldepth: 3}

func Tracef(message string, args ...any) {
	root.Tracef(message, args...)
}

func Debugf(message string, args ...any) {
	root.Debugf(message, args...)
}

func Infof(message string, args ...any) {
	root.Infof(message, args...)
}

func Warnf(message string, args ...any) {
	root.Warnf(message, args...)
}

func Errorf(message string, args ...any) {
	root.Errorf(message, args...)
}
