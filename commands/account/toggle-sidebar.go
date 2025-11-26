package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type ToggleSidebar struct{}

func init() {
	commands.Register(ToggleSidebar{})
}

func (ToggleSidebar) Description() string {
	return "Toggle the sidebar on or off."
}

func (ToggleSidebar) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (ToggleSidebar) Aliases() []string {
	return []string{"toggle-sidebar"}
}

func (ToggleSidebar) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	acct.ToggleSidebar()
	return nil
}
