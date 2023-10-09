package msgview

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/state"
)

type ToggleKeyPassthrough struct{}

func init() {
	register(ToggleKeyPassthrough{})
}

func (ToggleKeyPassthrough) Aliases() []string {
	return []string{"toggle-key-passthrough"}
}

func (ToggleKeyPassthrough) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (ToggleKeyPassthrough) Execute(aerc *app.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: toggle-key-passthrough")
	}
	mv, _ := aerc.SelectedTabContent().(*app.MessageViewer)
	keyPassthroughEnabled := mv.ToggleKeyPassthrough()
	if acct := mv.SelectedAccount(); acct != nil {
		acct.SetStatus(state.Passthrough(keyPassthroughEnabled))
	}
	return nil
}
