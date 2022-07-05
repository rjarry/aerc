package logging

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

var UICleanup = func() {}

// PanicHandler tries to restore the terminal. A stack trace is written to
// aerc-crash.log and then passed on if a panic occurs.
func PanicHandler() {
	r := recover()

	if r == nil {
		return
	}

	UICleanup()

	filename := time.Now().Format("/tmp/aerc-crash-20060102-150405.log")

	panicLog, err := os.OpenFile(filename, os.O_SYNC|os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		// we tried, not possible. bye
		panic(r)
	}
	defer panicLog.Close()

	outputs := io.MultiWriter(panicLog, os.Stderr)

	// if any error happens here, we do not care.
	fmt.Fprintln(panicLog, strings.Repeat("#", 80))
	fmt.Fprint(panicLog, strings.Repeat(" ", 34))
	fmt.Fprintln(panicLog, "PANIC CAUGHT!")
	fmt.Fprint(panicLog, strings.Repeat(" ", 24))
	fmt.Fprintln(panicLog, time.Now().Format("2006-01-02T15:04:05.000000-0700"))
	fmt.Fprintln(panicLog, strings.Repeat("#", 80))
	fmt.Fprintf(outputs, "%s\n", panicMessage)
	fmt.Fprintf(panicLog, "Error: %v\n\n", r)
	panicLog.Write(debug.Stack())
	fmt.Fprintf(os.Stderr, "\nThis error was also written to: %s\n", filename)
	panic(r)
}

const panicMessage = `
aerc has encountered a critical error and has terminated. Please help us fix
this by sending this log and the steps to reproduce the crash to:
~rjarry/aerc-devel@lists.sr.ht

Thank you
`
