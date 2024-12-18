package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type ExpandCollapseFolder struct {
	Folder string `opt:"folder" required:"false" complete:"CompleteFolder" desc:"Folder name."`
}

func init() {
	commands.Register(ExpandCollapseFolder{})
}

func (ExpandCollapseFolder) Description() string {
	return "Expand or collapse the current folder."
}

func (ExpandCollapseFolder) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (ExpandCollapseFolder) Aliases() []string {
	return []string{"expand-folder", "collapse-folder"}
}

func (*ExpandCollapseFolder) CompleteFolder(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return nil
	}
	return commands.FilterList(acct.Directories().List(), arg, nil)
}

func (e ExpandCollapseFolder) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if e.Folder == "" {
		e.Folder = acct.Directories().Selected()
	}
	if args[0] == "expand-folder" {
		acct.Directories().ExpandFolder(e.Folder)
	} else {
		acct.Directories().CollapseFolder(e.Folder)
	}
	return nil
}
