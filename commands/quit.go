package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Quit struct{}

func init() {
	register(Quit{})
}

func (_ Quit) Aliases() []string {
	return []string{"quit", "exit"}
}

func (_ Quit) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

type ErrorExit int

func (err ErrorExit) Error() string {
	return "exit"
}

func (_ Quit) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: quit")
	}
	return ErrorExit(1)
}
