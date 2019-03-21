package commands

var (
	GlobalCommands *Commands
)

func register(name string, cmd AercCommand) {
	if GlobalCommands == nil {
		GlobalCommands = NewCommands()
	}
	GlobalCommands.Register(name, cmd)
}
