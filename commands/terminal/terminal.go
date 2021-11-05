package terminal

import (
	"git.sr.ht/~rjarry/aerc/commands"
)

var (
	TerminalCommands *commands.Commands
)

func register(cmd commands.Command) {
	if TerminalCommands == nil {
		TerminalCommands = commands.NewCommands()
	}
	TerminalCommands.Register(cmd)
}
