package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	Register("quit", ChangeQuit)
}

type ErrorExit int

func (err ErrorExit) Error() string {
	return "exit"
}

func ChangeQuit(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: quit")
	}
	return ErrorExit(1)
}
