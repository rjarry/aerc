package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
)

type ExpandCollapseFolder struct{}

func init() {
	register(ExpandCollapseFolder{})
}

func (ExpandCollapseFolder) Aliases() []string {
	return []string{"expand-folder", "collapse-folder"}
}

func (ExpandCollapseFolder) Complete(args []string) []string {
	return nil
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
