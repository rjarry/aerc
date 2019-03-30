package terminal

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	register("close", CommandClose)
}

func CommandClose(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: close")
	}
	term, ok := aerc.SelectedTab().(*widgets.Terminal)
	if !ok {
		return errors.New("Error: not a terminal")
	}
	term.Close(nil)
	aerc.RemoveTab(term)
	return nil
}
