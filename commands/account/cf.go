package account

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

var (
	history map[string]string
)

type ChangeFolder struct{}

func init() {
	history = make(map[string]string)
	register(ChangeFolder{})
}

func (_ ChangeFolder) Aliases() []string {
	return []string{"cf"}
}

func (_ ChangeFolder) Complete(aerc *widgets.Aerc, args []string) []string {
	return commands.GetFolders(aerc, args)
}

func (_ ChangeFolder) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: cf <folder>")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	previous := acct.Directories().Selected()
	if args[1] == "-" {
		if dir, ok := history[acct.Name()]; ok {
			store.ApplyClear()
			acct.Directories().Select(dir)
		} else {
			return errors.New("No previous folder to return to")
		}
	} else {
		store.ApplyClear()
		acct.Directories().Select(args[1])
	}
	history[acct.Name()] = previous
	return nil
}
