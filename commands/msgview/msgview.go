package msgview

import (
	"git.sr.ht/~sircmpwn/aerc2/commands"
)

var (
	MessageViewCommands *commands.Commands
)

func register(name string, cmd commands.AercCommand) {
	if MessageViewCommands == nil {
		MessageViewCommands = commands.NewCommands()
	}
	MessageViewCommands.Register(name, cmd)
}
