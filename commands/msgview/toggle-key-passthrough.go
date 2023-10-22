package msgview

import (
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

func (ToggleKeyPassthrough) Execute(args []string) error {
	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
	keyPassthroughEnabled := mv.ToggleKeyPassthrough()
	if acct := mv.SelectedAccount(); acct != nil {
		acct.SetStatus(state.Passthrough(keyPassthroughEnabled))
	}
	return nil
}
