package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

type AercCommand func(aerc *widgets.Aerc, cmd string) error

var (
	commands map[string]AercCommand
)

func init() {
	commands = make(map[string]AercCommand)
}

func Register(name string, cmd AercCommand) {
	commands[name] = cmd
}

func ExecuteCommand(aerc *widgets.Aerc, cmd string) error {
	if fn, ok := commands[cmd]; ok {
		return fn(aerc, cmd)
	}
	return errors.New("Unknown command " + cmd)
}
