package compose

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type NextPrevField struct{}

func init() {
	commands.Register(NextPrevField{})
}

func (NextPrevField) Description() string {
	return "Cycle between header input fields."
}

func (NextPrevField) Context() commands.CommandContext {
	return commands.COMPOSE
}

func (NextPrevField) Aliases() []string {
	return []string{"next-field", "prev-field"}
}

func (NextPrevField) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	var ok bool
	if args[0] == "prev-field" {
		ok = composer.PrevField()
	} else {
		ok = composer.NextField()
	}
	if !ok {
		return fmt.Errorf("%s not available when edit-headers=true", args[0])
	}
	return nil
}
