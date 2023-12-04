package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type ExpandCollapseFolder struct{}

func init() {
	commands.Register(ExpandCollapseFolder{})
}

func (ExpandCollapseFolder) Context() commands.CommandContext {
	return commands.ACCOUNT
}

func (ExpandCollapseFolder) Aliases() []string {
	return []string{"expand-folder", "collapse-folder"}
}

func (ExpandCollapseFolder) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if args[0] == "expand-folder" {
		acct.Directories().ExpandFolder()
	} else {
		acct.Directories().CollapseFolder()
	}
	return nil
}
