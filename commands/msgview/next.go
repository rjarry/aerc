package msgview

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type NextPrevMsg struct{}

func init() {
	register(NextPrevMsg{})
}

func (_ NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (_ NextPrevMsg) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ NextPrevMsg) Execute(aerc *widgets.Aerc, args []string) error {
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
