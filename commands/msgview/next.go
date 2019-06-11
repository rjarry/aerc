package msgview

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("next", NextPrevMessage)
	register("next-message", NextPrevMessage)
	register("prev", NextPrevMessage)
	register("prev-message", NextPrevMessage)
}

func NextPrevMessage(aerc *widgets.Aerc, args []string) error {
	mv, _ := aerc.SelectedTab().(*widgets.MessageViewer)
	acct := mv.SelectedAccount()
	store := mv.Store()
	if acct == nil {
		return errors.New("No account selected")
	}
	if args[0] == "prev-message" || args[0] == "prev" {
		store.Prev()
	} else {
		store.Next()
	}
	nextMsg := store.Selected()
	if nextMsg == nil {
		aerc.RemoveTab(mv)
		return nil
	}
	nextMv := widgets.NewMessageViewer(acct, aerc.Config(), store, nextMsg)
	aerc.ReplaceTab(mv, nextMv, nextMsg.Envelope.Subject)
	return nil
}
