package msgview

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type ToggleHeaders struct{}

func init() {
	register(ToggleHeaders{})
}

func (ToggleHeaders) Aliases() []string {
	return []string{"toggle-headers"}
}

func (ToggleHeaders) Execute(args []string) error {
	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
	mv.ToggleHeaders()
	return nil
}
