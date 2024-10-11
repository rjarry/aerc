package msgview

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/state"
)

type ToggleKeyPassthrough struct{}

func init() {
	commands.Register(ToggleKeyPassthrough{})
}

func (ToggleKeyPassthrough) Description() string {
	return "Enter or exit the passthrough key bindings context."
}

func (ToggleKeyPassthrough) Context() commands.CommandContext {
	return commands.MESSAGE_VIEWER
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
