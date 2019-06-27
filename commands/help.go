package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Help struct{}

func init() {
	register(Help{})
}

func (_ Help) Aliases() []string {
	return []string{"help"}
}

func (_ Help) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Help) Execute(aerc *widgets.Aerc, args []string) error {
	page := "aerc"
	if len(args) == 2 {
		page = "aerc-" + args[1]
	} else if len(args) > 2 {
		return errors.New("Usage: help [topic]")
	}
	return TermCore(aerc, []string{"term", "man", page})
}
