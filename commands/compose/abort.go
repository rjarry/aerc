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

func (Abort) Complete(args []string) []string {
	return nil
}

func (Abort) Execute(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: abort")
	}
	composer, _ := app.SelectedTabContent().(*app.Composer)

	app.RemoveTab(composer, true)

	return nil
}
