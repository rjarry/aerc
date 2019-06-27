package commands

var (
	GlobalCommands *Commands
)

func register(cmd Command) {
	if GlobalCommands == nil {
		GlobalCommands = NewCommands()
	}
	GlobalCommands.Register(cmd)
}
