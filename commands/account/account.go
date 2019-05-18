package account

import (
	"git.sr.ht/~sircmpwn/aerc/commands"
)

var (
	AccountCommands *commands.Commands
)

func register(name string, cmd commands.AercCommand) {
	if AccountCommands == nil {
		AccountCommands = commands.NewCommands()
	}
	AccountCommands.Register(name, cmd)
}
