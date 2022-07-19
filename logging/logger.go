package logging

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var (
	dbg  *log.Logger
	info *log.Logger
	warn *log.Logger
	err  *log.Logger
)

func Init() {
	flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC
	dbg = log.New(os.Stdout, "DEBUG ", flags)
	info = log.New(os.Stdout, "INFO  ", flags)
	warn = log.New(os.Stdout, "WARN  ", flags)
	err = log.New(os.Stdout, "ERROR ", flags)
}

func ErrorLogger() *log.Logger {
	if err == nil {
		return log.New(ioutil.Discard, "", log.LstdFlags)
	}
	return err
}

func Debugf(message string, args ...interface{}) {
	if dbg == nil {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	dbg.Output(2, message)
}

func Infof(message string, args ...interface{}) {
	if info == nil {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	info.Output(2, message)
}

func Warnf(message string, args ...interface{}) {
	if warn == nil {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	warn.Output(2, message)
}

func Errorf(message string, args ...interface{}) {
	if err == nil {
		return
	}
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	err.Output(2, message)
}
