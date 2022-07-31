package commands

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/commands/mode"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Quit struct{}

func init() {
	register(Quit{})
}

func (Quit) Aliases() []string {
	return []string{"quit", "exit"}
}

func (Quit) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

type ErrorExit int

func (err ErrorExit) Error() string {
	return "exit"
}

func (Quit) Execute(aerc *widgets.Aerc, args []string) error {
	force := false
	opts, optind, err := getopt.Getopts(args, "f")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		if opt.Option == 'f' {
			force = true
		}
	}
	if len(args) != optind {
		return errors.New("Usage: quit [-f]")
	}
	if force || mode.QuitAllowed() {
		return ErrorExit(1)
	}
	return fmt.Errorf("A task is not done yet. Use -f to force an exit.")
}
