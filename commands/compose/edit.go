package compose

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Edit struct{}

func init() {
	register(Edit{})
}

func (_ Edit) Aliases() []string {
	return []string{"edit"}
}

func (_ Edit) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Edit) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: edit")
	}
	composer, _ := aerc.SelectedTab().(*widgets.Composer)
	composer.ShowTerminal()
	composer.FocusTerminal()
	return nil
}
