package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("quit", CommandQuit)
}

type ErrorExit int

func (err ErrorExit) Error() string {
	return "exit"
}

func CommandQuit(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: quit")
	}
	return ErrorExit(1)
}
