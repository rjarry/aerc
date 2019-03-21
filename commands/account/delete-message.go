package account

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	register("delete-message", DeleteMessage)
}

func DeleteMessage(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: :delete-message")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Messages().Store()
	msg := acct.Messages().Selected()
	store.Delete([]uint32{msg.Uid})
	return nil
}
