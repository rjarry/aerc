package commands

import (
	"errors"

	"github.com/google/shlex"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

type AercCommand func(aerc *widgets.Aerc, args []string) error

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
	args, err := shlex.Split(cmd)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("Expected a command.")
	}
	if fn, ok := commands[args[0]]; ok {
		return fn(aerc, args)
	}
	return errors.New("Unknown command " + args[0])
}
