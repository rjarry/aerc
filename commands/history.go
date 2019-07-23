package commands

type cmdHistory struct {
	// rolling buffer of prior commands
	//
	// most recent command is at the end of the list,
	// least recent is index 0
	cmdList []string

	// current placement in list
	current int
}

// number of commands to keep in history
const cmdLimit = 1000

// CmdHistory is the history of executed commands
var CmdHistory = cmdHistory{}

func (h *cmdHistory) Add(cmd string) {
	// if we're at cap, cut off the first element
	if len(h.cmdList) >= cmdLimit {
		h.cmdList = h.cmdList[1:]
	}

	h.cmdList = append(h.cmdList, cmd)

	// whenever we add a new command, reset the current
	// pointer to the "beginning" of the list
	h.Reset()
}

// Prev returns the previous command in history.
// Since the list is reverse-order, this will return elements
// increasingly towards index 0.
func (h *cmdHistory) Prev() string {
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
