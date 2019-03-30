package account

import (
	"errors"

	"github.com/mohamedattahri/mail"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	register("fetch-message", FetchMessage)
}

func FetchMessage(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: :fetch-message")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Messages().Store()
	msg := acct.Messages().Selected()
	store.FetchBodies([]uint32{msg.Uid}, func(msg *mail.Message) {
		aerc.SetStatus("got message body, woohoo")
	})
	return nil
}
