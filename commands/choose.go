package commands

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
)

type Choose struct{}

func init() {
	register(Choose{})
}

func (Choose) Aliases() []string {
	return []string{"choose"}
}

func (Choose) Complete(args []string) []string {
	return nil
}

func (Choose) Execute(args []string) error {
	if len(args) < 5 || len(args)%4 != 1 {
		return chooseUsage(args[0])
	}

	choices := []app.Choice{}
	for i := 0; i+4 < len(args); i += 4 {
		if args[i+1] != "-o" {
			return chooseUsage(args[0])
		}
		choices = append(choices, app.Choice{
			Key:     args[i+2],
			Text:    args[i+3],
			Command: args[i+4],
		})
	}

	app.RegisterChoices(choices)

	return nil
}

func chooseUsage(cmd string) error {
	return fmt.Errorf("Usage: %s -o <key> <text> <command> [-o <key> <text> <command>]...", cmd)
}
