package terminal

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Close struct{}

func init() {
	register(Close{})
}

func (_ Close) Aliases() []string {
	return []string{"close"}
}

func (_ Close) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Close) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: close")
	}
	term, _ := aerc.SelectedTab().(*widgets.Terminal)
	term.Close(nil)
	return nil
}
