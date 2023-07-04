package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Edit struct{}

func init() {
	register(Edit{})
}

func (Edit) Aliases() []string {
	return []string{"edit"}
}

func (Edit) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Edit) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: edit")
	}
	composer, ok := aerc.SelectedTabContent().(*widgets.Composer)
	if !ok {
		return errors.New("only valid while composing")
	}
	err := composer.ShowTerminal()
	if err != nil {
		return err
	}
	composer.FocusTerminal()
	return nil
}
