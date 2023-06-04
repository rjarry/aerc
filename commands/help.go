package commands

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Help struct{}

var pages = []string{
	"aerc",
	"accounts",
	"binds",
	"config",
	"imap",
	"jmap",
	"notmuch",
	"search",
	"sendmail",
	"smtp",
	"stylesets",
	"templates",
	"tutorial",
	"keys",
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

	if page == "aerc-keys" {
		aerc.AddDialog(widgets.NewDialog(
			widgets.NewListBox(
				"Bindings: Press <Esc> or <Enter> to close. "+
					"Start typing to filter bindings.",
				aerc.HumanReadableBindings(),
				aerc.SelectedAccountUiConfig(),
				func(_ string) {
					aerc.CloseDialog()
				},
			),
			func(h int) int { return h / 4 },
			func(h int) int { return h / 2 },
		))
		return nil
	}

	return TermCore(aerc, []string{"term", "man", page})
}
