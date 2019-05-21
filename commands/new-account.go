package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("new-account", CommandNewAccount)
}

func CommandNewAccount(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: new-account")
	}
	wizard := widgets.NewAccountWizard()
	aerc.NewTab(wizard, "New account")
	return nil
}
