package compose

import (
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
	composer, _ := app.SelectedTabContent().(*app.Composer)
	app.RemoveTab(composer, true)
	return nil
}
