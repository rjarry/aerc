package compose

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("edit", CommandEdit)
}

func CommandEdit(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: edit")
	}
	composer, _ := aerc.SelectedTab().(*widgets.Composer)
	composer.ShowTerminal()
	composer.FocusTerminal()
	return nil
}
