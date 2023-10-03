package msgview

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

func (Close) Complete(args []string) []string {
	return nil
}

func (Close) Execute(args []string) error {
	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
	app.RemoveTab(mv, true)
	return nil
}
