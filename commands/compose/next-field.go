package compose

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type NextPrevField struct{}

func init() {
	commands.Register(NextPrevField{})
}

func (NextPrevField) Context() commands.CommandContext {
	return commands.COMPOSE
}

func (NextPrevField) Aliases() []string {
	return []string{"next-field", "prev-field"}
}

func (NextPrevField) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	if args[0] == "prev-field" {
		composer.PrevField()
	} else {
		composer.NextField()
	}
	return nil
}
