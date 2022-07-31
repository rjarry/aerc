package compose

import (
	"git.sr.ht/~rjarry/aerc/commands"
)

var ComposeCommands *commands.Commands

func register(cmd commands.Command) {
	if ComposeCommands == nil {
		ComposeCommands = commands.NewCommands()
	}
	ComposeCommands.Register(cmd)
}
