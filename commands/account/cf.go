package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/state"
)

var history map[string]string

type ChangeFolder struct {
	Folder string `opt:"folder" complete:"CompleteFolder"`
}

func init() {
	history = make(map[string]string)
	register(ChangeFolder{})
}

func (ChangeFolder) Aliases() []string {
	return []string{"cf"}
}

func (*ChangeFolder) CompleteFolder(arg string) []string {
	return commands.GetFolders(arg)
}

func (c ChangeFolder) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	previous := acct.Directories().Selected()

	if c.Folder == "-" {
		if dir, ok := history[acct.Name()]; ok {
			acct.Directories().Select(dir)
		} else {
			return errors.New("No previous folder to return to")
		}
	} else {
		acct.Directories().Select(c.Folder)
	}
	history[acct.Name()] = previous

	// reset store filtering if we switched folders
	store := acct.Store()
	if store != nil {
		store.ApplyClear()
		acct.SetStatus(state.SearchFilterClear())
	}
	return nil
}
