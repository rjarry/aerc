package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type Repeat struct{}

func init() {
	Register(Repeat{})
}

func (Repeat) Description() string {
	return "Repeat aerc's last executed command."
}

func (Repeat) Context() CommandContext {
	return GLOBAL
}

func (Repeat) Aliases() []string {
	return []string{"repeat"}
}

func (Repeat) Execute(args []string) error {
	return app.ExecuteLastCommand()
}
