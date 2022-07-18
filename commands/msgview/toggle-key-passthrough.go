package msgview

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/lib/statusline"
	"git.sr.ht/~rjarry/aerc/widgets"
)

type ToggleKeyPassthrough struct{}

func init() {
	register(ToggleKeyPassthrough{})
}

func (ToggleKeyPassthrough) Aliases() []string {
	return []string{"toggle-key-passthrough"}
}

func (ToggleKeyPassthrough) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (ToggleKeyPassthrough) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: toggle-key-passthrough")
	}
	mv, _ := aerc.SelectedTabContent().(*widgets.MessageViewer)
	keyPassthroughEnabled := mv.ToggleKeyPassthrough()
	if acct := mv.SelectedAccount(); acct != nil {
		acct.SetStatus(statusline.Passthrough(keyPassthroughEnabled))
	}
	return nil
}
