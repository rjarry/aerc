package parse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"

	"git.sr.ht/~rjarry/aerc/logging"
)

var ansi = regexp.MustCompile("\x1B\\[[0-?]*[ -/]*[@-~]")

// StripAnsi strips ansi escape codes from the reader
func StripAnsi(r io.Reader) io.Reader {
	buf := bytes.NewBuffer(nil)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, 1024*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		line = ansi.ReplaceAll(line, []byte(""))
		_, err := buf.Write(line)
		if err != nil {
			logging.Warnf("failed write ", err)
		}
		_, err = buf.Write([]byte("\n"))
		if err != nil {
			logging.Warnf("failed write ", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read line: %v\n", err)
	}
	return buf
}
