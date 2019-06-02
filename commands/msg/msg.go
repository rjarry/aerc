package msg

import (
	"git.sr.ht/~sircmpwn/aerc/commands"
)

var (
	MessageCommands *commands.Commands
)

func register(name string, cmd commands.AercCommand) {
	if MessageCommands == nil {
		MessageCommands = commands.NewCommands()
	}
	MessageCommands.Register(name, cmd)
}
