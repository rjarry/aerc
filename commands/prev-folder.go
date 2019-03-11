package commands

import (
	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	Register("prev-folder", PrevFolder)
}

func PrevFolder(aerc *widgets.Aerc, cmd string) error {
	acct := aerc.SelectedAccount()
	acct.Directories().Prev()
	return nil
}
