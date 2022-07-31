package account

import (
	"git.sr.ht/~rjarry/aerc/commands"
)

var AccountCommands *commands.Commands

func register(cmd commands.Command) {
	if AccountCommands == nil {
		AccountCommands = commands.NewCommands()
	}
	AccountCommands.Register(cmd)
}
