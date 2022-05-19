package commands

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Help struct{}

var pages = []string{
	"aerc",
	"config",
	"imap",
	"notmuch",
	"search",
	"sendmail",
	"smtp",
	"stylesets",
	"templates",
	"tutorial",
}

func init() {
	register(Help{})
}

func (Help) Aliases() []string {
	return []string{"help"}
}

func (Help) Complete(aerc *widgets.Aerc, args []string) []string {
	return CompletionFromList(aerc, pages, args)
}

func (Help) Execute(aerc *widgets.Aerc, args []string) error {
	page := "aerc"
	if len(args) == 2 && args[1] != "aerc" {
		page = "aerc-" + args[1]
	} else if len(args) > 2 {
		return errors.New("Usage: help [topic]")
	}
	return TermCore(aerc, []string{"term", "man", page})
}
