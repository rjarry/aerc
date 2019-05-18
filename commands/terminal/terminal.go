package terminal

import (
	"git.sr.ht/~sircmpwn/aerc/commands"
)

var (
	TerminalCommands *commands.Commands
)

func register(name string, cmd commands.AercCommand) {
	if TerminalCommands == nil {
		TerminalCommands = commands.NewCommands()
	}
	TerminalCommands.Register(name, cmd)
}
