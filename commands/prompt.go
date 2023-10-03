package commands

import (
	"strings"

	"git.sr.ht/~rjarry/go-opt"

	"git.sr.ht/~rjarry/aerc/app"
)

type Prompt struct {
	Text string   `opt:"text"`
	Cmd  []string `opt:"..."`
}

func init() {
	register(Prompt{})
}

func (Prompt) Aliases() []string {
	return []string{"prompt"}
}

func (Prompt) Complete(args []string) []string {
	argc := len(args)
	if argc == 0 {
		return nil
	}
	hascommand := argc > 2
	if argc == 1 {
		args = append(args, "")
	}

	cmd := GlobalCommands.ByName(args[1])
	var cs []string
	if cmd != nil {
		cs = cmd.Complete(args[2:])
		hascommand = true
	} else {
		if hascommand {
			return nil
		}
		cs, _ = GlobalCommands.GetCompletions(args[1])
	}
	if cs == nil {
		return nil
	}

	var b strings.Builder
	// it seems '' quoting is enough
	// to keep quoted arguments in one piece
	b.WriteRune('\'')
	b.WriteString(args[0])
	b.WriteRune('\'')
	b.WriteRune(' ')
	if hascommand {
		b.WriteString(args[1])
		b.WriteRune(' ')
	}

	src := b.String()
	b.Reset()

	rs := make([]string, 0, len(cs))
	for _, c := range cs {
		b.WriteString(src)
		b.WriteString(c)

		rs = append(rs, b.String())
		b.Reset()
	}

	return rs
}

func (p Prompt) Execute(args []string) error {
	cmd := opt.QuoteArgs(p.Cmd...)
	app.RegisterPrompt(p.Text, cmd.String())
	return nil
}
