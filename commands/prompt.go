package commands

import (
	"fmt"
	"git.sr.ht/~rjarry/aerc/widgets"
	"strings"
)

type Prompt struct{}

func init() {
	register(Prompt{})
}

func (Prompt) Aliases() []string {
	return []string{"prompt"}
}

func (Prompt) Complete(aerc *widgets.Aerc, args []string) []string {
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
		cs = cmd.Complete(aerc, args[2:])
		hascommand = true
	} else {
		if hascommand {
			return nil
		}
		cs = GlobalCommands.GetCompletions(aerc, args[1])
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

func (Prompt) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("Usage: %s <prompt> <cmd>", args[0])
	}

	prompt := args[1]
	cmd := args[2:]
	aerc.RegisterPrompt(prompt, cmd)
	return nil
}
