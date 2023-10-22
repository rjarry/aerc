package terminal

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type Close struct{}

func init() {
	register(Close{})
}

func (Close) Aliases() []string {
	return []string{"close"}
}

func (Close) Execute(args []string) error {
	term, _ := app.SelectedTabContent().(*app.Terminal)
	term.Close()
	return nil
}
