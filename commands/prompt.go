package commands

import (
	"git.sr.ht/~rjarry/go-opt"

	"git.sr.ht/~rjarry/aerc/app"
)

type Prompt struct {
	Text string   `opt:"text"`
	Cmd  []string `opt:"..." complete:"CompleteCommand"`
}

func init() {
	register(Prompt{})
}

func (Prompt) Aliases() []string {
	return []string{"prompt"}
}

func (*Prompt) CompleteCommand(arg string) []string {
	return FilterList(GlobalCommands.Names(), arg, nil)
}

func (p Prompt) Execute(args []string) error {
	cmd := opt.QuoteArgs(p.Cmd...)
	app.RegisterPrompt(p.Text, cmd.String())
	return nil
}
