package terminal

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
)

type Close struct{}

func init() {
	register(Close{})
}

func (Close) Aliases() []string {
	return []string{"close"}
}

func (Close) Complete(args []string) []string {
	return nil
}

func (Close) Execute(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: close")
	}
	term, _ := app.SelectedTabContent().(*app.Terminal)
	term.Close()
	return nil
}
