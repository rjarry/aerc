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

func (ToggleHeaders) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (ToggleHeaders) Execute(aerc *app.Aerc, args []string) error {
	if len(args) > 1 {
		return toggleHeadersUsage(args[0])
	}
	mv, _ := aerc.SelectedTabContent().(*app.MessageViewer)
	mv.ToggleHeaders()
	return nil
}

func toggleHeadersUsage(cmd string) error {
	return fmt.Errorf("Usage: %s", cmd)
}
