package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib"
)

type ViewMessage struct {
	Peek bool `opt:"-p"`
}

func init() {
	register(ViewMessage{})
}

func (ViewMessage) Aliases() []string {
	return []string{"view-message", "view"}
}

func (ViewMessage) Complete(args []string) []string {
	return nil
}

func (v ViewMessage) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
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
		app.PushError(msg.Error.Error())
		return nil
	}
	lib.NewMessageStoreView(msg, !v.Peek && acct.UiConfig().AutoMarkRead,
		store, app.CryptoProvider(), app.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				app.PushError(err.Error())
				return
			}
			viewer := app.NewMessageViewer(acct, view)
			app.NewTab(viewer, msg.Envelope.Subject)
		})
	return nil
}
