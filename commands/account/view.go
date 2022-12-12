package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
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
	peek := false
	opts, optind, err := getopt.Getopts(args, "p")
	if err != nil {
		return err
	}

	for _, opt := range opts {
		if opt.Option == 'p' {
			peek = true
		}
	}

	if len(args) != optind {
		return errors.New("Usage: view-message [-p]")
	}
	acct := aerc.SelectedAccount()
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
		aerc.PushError(msg.Error.Error())
		return nil
	}
	lib.NewMessageStoreView(msg, !peek && acct.UiConfig().AutoMarkRead,
		store, aerc.Crypto, aerc.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				aerc.PushError(err.Error())
				return
			}
			viewer := widgets.NewMessageViewer(acct, view)
			aerc.NewTab(viewer, msg.Envelope.Subject)
		})
	return nil
}
