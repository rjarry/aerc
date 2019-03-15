package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

var (
	history map[string]string
)

func init() {
	history = make(map[string]string)
	Register("cd", ChangeDirectory)
}

func ChangeDirectory(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: cd <directory>")
	}
	acct := aerc.SelectedAccount()
	previous := acct.Directories().Selected()
	if args[1] == "-" {
		if dir, ok := history[acct.Name()]; ok {
			acct.Directories().Select(dir)
		} else {
			return errors.New("No previous directory to return to")
		}
	} else {
		acct.Directories().Select(args[1])
	}
	history[acct.Name()] = previous
	return nil
}
