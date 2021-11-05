package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Abort struct{}

func init() {
	register(Abort{})
}

func (Abort) Aliases() []string {
	return []string{"abort"}
}

func (Abort) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Abort) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: abort")
	}
	composer, _ := aerc.SelectedTab().(*widgets.Composer)

	aerc.RemoveTab(composer)
	composer.Close()

	return nil
}
