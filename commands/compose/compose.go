package compose

import (
	"git.sr.ht/~sircmpwn/aerc/commands"
)

var (
	ComposeCommands *commands.Commands
)

func register(name string, cmd commands.AercCommand) {
	if ComposeCommands == nil {
		ComposeCommands = commands.NewCommands()
	}
	ComposeCommands.Register(name, cmd)
}
