package compose

import (
	"git.sr.ht/~sircmpwn/aerc/commands"
)

var (
	ComposeCommands *commands.Commands
)

func register(cmd commands.Command) {
	if ComposeCommands == nil {
		ComposeCommands = commands.NewCommands()
	}
	ComposeCommands.Register(cmd)
}
