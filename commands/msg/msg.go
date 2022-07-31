package msg

import (
	"git.sr.ht/~rjarry/aerc/commands"
)

var MessageCommands *commands.Commands

func register(cmd commands.Command) {
	if MessageCommands == nil {
		MessageCommands = commands.NewCommands()
	}
	MessageCommands.Register(cmd)
}
