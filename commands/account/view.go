package account

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type ViewMessage struct{}

func init() {
	register(ViewMessage{})
}

func (ViewMessage) Aliases() []string {
	return []string{"view-message", "view"}
}

func (ViewMessage) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (ViewMessage) Execute(aerc *widgets.Aerc, args []string) error {
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
	_, deleted := store.Deleted[msg.Uid]
	if deleted {
		return nil
	}
	viewer := widgets.NewMessageViewer(acct, aerc.Config(), store, msg)
	aerc.NewTab(viewer, msg.Envelope.Subject)
	return nil
}
