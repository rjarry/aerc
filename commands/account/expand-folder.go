package account

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
)

type ExpandCollapseFolder struct{}

func init() {
	register(ExpandCollapseFolder{})
}

func (ExpandCollapseFolder) Aliases() []string {
	return []string{"expand-folder", "collapse-folder"}
}

func (ExpandCollapseFolder) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (ExpandCollapseFolder) Execute(aerc *app.Aerc, args []string) error {
	if len(args) > 1 {
		return expandCollapseFolderUsage(args[0])
	}
	acct := aerc.SelectedAccount()
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

func expandCollapseFolderUsage(cmd string) error {
	return fmt.Errorf("Usage: %s", cmd)
}
