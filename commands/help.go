package commands

import (
	"fmt"
	"slices"

	"git.sr.ht/~rjarry/aerc/app"
)

type Help struct {
	Topic string `opt:"topic" action:"ParseTopic" default:"aerc" complete:"CompleteTopic" desc:"Help topic."`
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
	"patch",
	"keys",
}

func init() {
	Register(Help{})
}

func (Help) Description() string {
	return "Display one of aerc's man pages in the embedded terminal."
}

func (Help) Context() CommandContext {
	return GLOBAL
}

func (Help) Aliases() []string {
	return []string{"help", "man"}
}

func (*Help) CompleteTopic(arg string) []string {
	return FilterList(pages, arg, nil)
}

func (h *Help) ParseTopic(arg string) error {
	if slices.Contains(pages, arg) {
		if arg != "aerc" {
			arg = "aerc-" + arg
		}
		h.Topic = arg
		return nil
	}
	return fmt.Errorf("unknown topic %q", arg)
}

func (h Help) Execute(args []string) error {
	if h.Topic == "aerc-keys" {
		app.AddDialog(app.DefaultDialog(
			app.NewListBox(
				"Bindings: Press <Esc> or <Enter> to close. "+
					"Start typing to filter bindings.",
				app.HumanReadableBindings(),
				app.SelectedAccountUiConfig(),
				func(_ string) {
					app.CloseDialog()
				},
			),
		))
		return nil
	}
	term := Term{Cmd: []string{"man", h.Topic}}
	return term.Execute(args)
}
