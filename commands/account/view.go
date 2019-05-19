package account

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("view", ViewMessage)
	register("view-message", ViewMessage)
}

func ViewMessage(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: view-message")
	}
	acct := aerc.SelectedAccount()
	if acct.Messages().Empty() {
		return nil
	}
	store := acct.Messages().Store()
	msg := acct.Messages().Selected()
	if msg == nil {
		return nil
	}
	viewer := widgets.NewMessageViewer(aerc.Config(), store, msg)
	aerc.NewTab(viewer, msg.Envelope.Subject)
	return nil
}
