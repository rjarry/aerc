package msgview

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
)

type ToggleHeaders struct{}

func init() {
	register(ToggleHeaders{})
}

func (ToggleHeaders) Aliases() []string {
	return []string{"toggle-headers"}
}

func (ToggleHeaders) Complete(args []string) []string {
	return nil
}

func (ToggleHeaders) Execute(args []string) error {
	if len(args) > 1 {
		return toggleHeadersUsage(args[0])
	}
	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
	mv.ToggleHeaders()
	return nil
}

func toggleHeadersUsage(cmd string) error {
	return fmt.Errorf("Usage: %s", cmd)
}
