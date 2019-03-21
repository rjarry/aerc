package account

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

var (
	history map[string]string
)

func init() {
	history = make(map[string]string)
	register("cf", ChangeFolder)
}

func ChangeFolder(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: cf <folder>")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	previous := acct.Directories().Selected()
	if args[1] == "-" {
		if dir, ok := history[acct.Name()]; ok {
			acct.Directories().Select(dir)
		} else {
			return errors.New("No previous folder to return to")
		}
	} else {
		acct.Directories().Select(args[1])
	}
	history[acct.Name()] = previous
	return nil
}
