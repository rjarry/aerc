package account

import (
	"errors"
	"io"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("pipe", Pipe)
}

func Pipe(aerc *widgets.Aerc, args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: :pipe <cmd> [args...]")
	}
	acct := aerc.SelectedAccount()
	store := acct.Messages().Store()
	msg := acct.Messages().Selected()
	store.FetchFull([]uint32{msg.Uid}, func(reader io.Reader) {
		term, err := commands.QuickTerm(aerc, args[1:], reader)
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}
		name := args[1] + " <" + msg.Envelope.Subject
		aerc.NewTab(term, name)
	})
	return nil
}
