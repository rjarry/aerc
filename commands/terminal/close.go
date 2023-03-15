package terminal

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Close struct{}

func init() {
	register(Close{})
}

func (Close) Aliases() []string {
	return []string{"close"}
}

func (Close) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Close) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: close")
	}
	term, _ := aerc.SelectedTabContent().(*widgets.Terminal)
	term.Close()
	return nil
}
