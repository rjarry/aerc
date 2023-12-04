package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
)

type Edit struct {
	Edit   bool `opt:"-e"`
	NoEdit bool `opt:"-E"`
}

func init() {
	commands.Register(Edit{})
}

func (Edit) Context() commands.CommandContext {
	return commands.COMPOSE
}

func (Edit) Aliases() []string {
	return []string{"edit"}
}

func (e Edit) Execute(args []string) error {
	composer, ok := app.SelectedTabContent().(*app.Composer)
	if !ok {
		return errors.New("only valid while composing")
	}

	editHeaders := (config.Compose.EditHeaders || e.Edit) && !e.NoEdit

	err := composer.ShowTerminal(editHeaders)
	if err != nil {
		return err
	}
	composer.FocusTerminal()
	return nil
}
