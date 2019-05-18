package commands

import (
	"errors"

	"github.com/google/shlex"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type AercCommand func(aerc *widgets.Aerc, args []string) error

type Commands map[string]AercCommand

func NewCommands() *Commands {
	cmds := Commands(make(map[string]AercCommand))
	return &cmds
}

func (cmds *Commands) dict() map[string]AercCommand {
	return map[string]AercCommand(*cmds)
}

func (cmds *Commands) Register(name string, cmd AercCommand) {
	cmds.dict()[name] = cmd
}

type NoSuchCommand string

func (err NoSuchCommand) Error() string {
	return "Unknown command " + string(err)
}

type CommandSource interface {
	Commands() *Commands
}

func (cmds *Commands) ExecuteCommand(aerc *widgets.Aerc, cmd string) error {
	args, err := shlex.Split(cmd)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("Expected a command.")
	}
	if fn, ok := cmds.dict()[args[0]]; ok {
		return fn(aerc, args)
	}
	return NoSuchCommand(args[0])
}
