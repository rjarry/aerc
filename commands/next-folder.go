package commands

import (
	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	Register("next-folder", NextFolder)
}

func NextFolder(aerc *widgets.Aerc, cmd string) error {
	acct := aerc.SelectedAccount()
	acct.Directories().Next()
	return nil
}
