package msgview

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

func (Close) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (Close) Execute(aerc *app.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: close")
	}
	mv, _ := aerc.SelectedTabContent().(*app.MessageViewer)
	aerc.RemoveTab(mv, true)
	return nil
}
