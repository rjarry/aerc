package msgview

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type ToggleHeaders struct{}

func init() {
	commands.Register(ToggleHeaders{})
}

func (ToggleHeaders) Description() string {
	return "Toggle the visibility of message headers."
}

func (ToggleHeaders) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (ToggleHeaders) Aliases() []string {
	return []string{"toggle-headers"}
}

func (ToggleHeaders) Execute(args []string) error {
	if commands.CurrentContext()&commands.MESSAGE_VIEWER > 0 {
		mv, _ := app.SelectedTabContent().(*app.MessageViewer)
		mv.ToggleHeaders()
	} else {
		acct := app.SelectedAccount()
		acct.ToggleHeaders()
	}
	return nil
}
