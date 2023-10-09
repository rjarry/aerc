package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
)

type Abort struct{}

func init() {
	register(Abort{})
}

func (Abort) Aliases() []string {
	return []string{"abort"}
}

func (Abort) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (Abort) Execute(aerc *app.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: abort")
	}
	composer, _ := aerc.SelectedTabContent().(*app.Composer)

	aerc.RemoveTab(composer, true)

	return nil
}
