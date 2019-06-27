package msg

import (
	"git.sr.ht/~sircmpwn/aerc/commands"
)

var (
	MessageCommands *commands.Commands
)

func register(cmd commands.Command) {
	if MessageCommands == nil {
		MessageCommands = commands.NewCommands()
	}
	MessageCommands.Register(cmd)
}
