package commands

import (
	"errors"
	"fmt"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Prompt struct{}

func init() {
	register(Prompt{})
}

func (_ Prompt) Aliases() []string {
	return []string{"prompt"}
}

func (_ Prompt) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil // TODO: add completions
}

func (_ Prompt) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 3 {
		return errors.New(fmt.Sprintf("Usage: %s <prompt> <cmd>", args[0]))
	}

	prompt := args[1]
	cmd := args[2:]
	aerc.RegisterPrompt(prompt, cmd)
	return nil
}
