package msgview

import (
	"git.sr.ht/~rjarry/aerc/commands"
)

var MessageViewCommands *commands.Commands

func register(cmd commands.Command) {
	if MessageViewCommands == nil {
		MessageViewCommands = commands.NewCommands()
	}
	MessageViewCommands.Register(cmd)
}
