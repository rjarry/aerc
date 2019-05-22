package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("help", Help)
}

func Help(aerc *widgets.Aerc, args []string) error {
	page := "aerc"
	if len(args) == 2 {
		page = "aerc-" + args[1]
	} else if len(args) > 2 {
		return errors.New("Usage: help [topic]")
	}
	return Term(aerc, []string{"term", "man", page})
}
