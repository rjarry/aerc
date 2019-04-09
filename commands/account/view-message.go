package account

import (
	"errors"

	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
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
	aerc.NewTab(viewer, runewidth.Truncate(
		msg.Envelope.Subject, 32, "…"))
	return nil
}

