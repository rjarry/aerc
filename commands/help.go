package commands

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
)

type Help struct {
	Topic string `opt:"topic" action:"ParseTopic" default:"aerc" complete:"CompleteTopic"`
}

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

func (*Help) CompleteTopic(arg string) []string {
	return FilterList(pages, arg, nil)
}

func (h *Help) ParseTopic(arg string) error {
	for _, page := range pages {
		if arg == page {
			if arg != "aerc" {
				arg = "aerc-" + arg
			}
			h.Topic = arg
			return nil
		}
	}
	return fmt.Errorf("unknown topic %q", arg)
}

func (h Help) Execute(args []string) error {
	if h.Topic == "aerc-keys" {
		app.AddDialog(app.NewDialog(
			app.NewListBox(
				"Bindings: Press <Esc> or <Enter> to close. "+
					"Start typing to filter bindings.",
				app.HumanReadableBindings(),
				app.SelectedAccountUiConfig(),
				func(_ string) {
					app.CloseDialog()
				},
			),
			func(h int) int { return h / 4 },
			func(h int) int { return h / 2 },
		))
		return nil
	}
	term := Term{Cmd: []string{"man", h.Topic}}
	return term.Execute(args)
}
