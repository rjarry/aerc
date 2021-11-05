package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/widgets"
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
	if msg.Error != nil {
		aerc.PushError(msg.Error.Error())
		return nil
	}
	lib.NewMessageStoreView(msg, store, aerc.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				aerc.PushError(err.Error())
				return
			}
			viewer := widgets.NewMessageViewer(acct, aerc.Config(), view)
			aerc.NewTab(viewer, msg.Envelope.Subject)
		})
	return nil
}
