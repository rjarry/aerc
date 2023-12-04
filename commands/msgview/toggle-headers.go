package msgview

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type ToggleHeaders struct{}

func init() {
	commands.Register(ToggleHeaders{})
}

func (ToggleHeaders) Context() commands.CommandContext {
	return commands.MESSAGE_VIEWER
}

func (ToggleHeaders) Aliases() []string {
	return []string{"toggle-headers"}
}

func (ToggleHeaders) Execute(args []string) error {
	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
	mv.ToggleHeaders()
	return nil
}
