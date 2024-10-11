package compose

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type Abort struct{}

func init() {
	commands.Register(Abort{})
}

func (Abort) Description() string {
	return "Close the composer without sending."
}

func (Abort) Context() commands.CommandContext {
	return commands.COMPOSE
}

func (Abort) Aliases() []string {
	return []string{"abort"}
}

func (Abort) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	app.RemoveTab(composer, true)
	return nil
}
