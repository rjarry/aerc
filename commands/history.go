package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

type cmdHistory struct {
	// rolling buffer of prior commands
	//
	// most recent command is at the end of the list,
	// least recent is index 0
	cmdList []string

	// current placement in list
	current int

	// initialize history storage
	initHistfile sync.Once
	histfile     io.ReadWriter
}

// number of commands to keep in history
const cmdLimit = 1000

// CmdHistory is the history of executed commands
var CmdHistory = cmdHistory{}

func (h *cmdHistory) Add(cmd string) {
	h.initHistfile.Do(h.initialize)

	// if we're at cap, cut off the first element
	if len(h.cmdList) >= cmdLimit {
		h.cmdList = h.cmdList[1:]
	}

	if len(h.cmdList) == 0 || h.cmdList[len(h.cmdList)-1] != cmd {
		h.cmdList = append(h.cmdList, cmd)

		h.writeHistory()
	}

	// whenever we add a new command, reset the current
	// pointer to the "beginning" of the list
	h.Reset()
}

// Prev returns the previous command in history.
// Since the list is reverse-order, this will return elements
// increasingly towards index 0.
func (h *cmdHistory) Prev() string {
	h.initHistfile.Do(h.initialize)

	if h.current <= 0 || len(h.cmdList) == 0 {
		h.current = -1
		return "(Already at beginning)"
	}
	h.current--

	return h.cmdList[h.current]
}

// Next returns the next command in history.
// Since the list is reverse-order, this will return elements
// increasingly towards index len(cmdList).
func (h *cmdHistory) Next() string {
	h.initHistfile.Do(h.initialize)

	if h.current >= len(h.cmdList)-1 || len(h.cmdList) == 0 {
		h.current = len(h.cmdList)
		return "(Already at end)"
	}
	h.current++

	return h.cmdList[h.current]
}

// Reset the current pointer to the beginning of history.
func (h *cmdHistory) Reset() {
	h.current = len(h.cmdList)
}

func (h *cmdHistory) initialize() {
	var err error
	openFlags := os.O_RDWR | os.O_EXCL

	histPath := xdg.StatePath("aerc", "history")
	if _, err := os.Stat(histPath); os.IsNotExist(err) {
		_ = os.MkdirAll(xdg.StatePath("aerc"), 0o700) // caught by OpenFile
		openFlags |= os.O_CREATE
	}

	// O_EXCL to make sure that only one aerc writes to the file
	h.histfile, err = os.OpenFile(
		histPath,
		openFlags,
		0o600,
	)
	if err != nil {
		log.Errorf("failed to open history file: %v", err)
		// basically mirror the old behavior
		h.histfile = bytes.NewBuffer([]byte{})
		return
	}

	s := bufio.NewScanner(h.histfile)

	for s.Scan() {
		h.cmdList = append(h.cmdList, s.Text())
	}

	h.Reset()
}

func (h *cmdHistory) writeHistory() {
	if fh, ok := h.histfile.(*os.File); ok {
		err := fh.Truncate(0)
		if err != nil {
			// if we can't delete it, don't break it.
			return
		}
		_, err = fh.Seek(0, io.SeekStart)
		if err != nil {
			// if we can't delete it, don't break it.
			return
		}
		for _, entry := range h.cmdList {
			fmt.Fprintln(fh, entry)
		}

		fh.Sync() //nolint:errcheck // if your computer can't sync you're in bigger trouble
	}
}
