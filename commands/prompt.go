package commands

import (
	"git.sr.ht/~rjarry/go-opt/v2"

	"git.sr.ht/~rjarry/aerc/app"
)

type Prompt struct {
	Text string   `opt:"text"`
	Cmd  []string `opt:"..." complete:"CompleteCommand"`
}

func init() {
	Register(Prompt{})
}

func (Prompt) Context() CommandContext {
	return GLOBAL
}

func (Prompt) Aliases() []string {
	return []string{"prompt"}
}

func (*Prompt) CompleteCommand(arg string) []string {
	return FilterList(ActiveCommandNames(), arg, nil)
}

func (p Prompt) Execute(args []string) error {
	cmd := opt.QuoteArgs(p.Cmd...)
	app.RegisterPrompt(p.Text, cmd.String())
	return nil
}
