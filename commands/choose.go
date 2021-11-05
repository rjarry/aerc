package commands

import (
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Choose struct{}

func init() {
	register(Choose{})
}

func (Choose) Aliases() []string {
	return []string{"choose"}
}

func (Choose) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Choose) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 5 || len(args)%4 != 1 {
		return chooseUsage(args[0])
	}

	choices := []widgets.Choice{}
	for i := 0; i+4 < len(args); i += 4 {
		if args[i+1] != "-o" {
			return chooseUsage(args[0])
		}
		choices = append(choices, widgets.Choice{
			Key:     args[i+2],
			Text:    args[i+3],
			Command: strings.Split(args[i+4], " "),
		})
	}

	aerc.RegisterChoices(choices)

	return nil
}

func chooseUsage(cmd string) error {
	return fmt.Errorf("Usage: %s -o <key> <text> <command> [-o <key> <text> <command>]...", cmd)
}
