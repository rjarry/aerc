package msgview

import (
	"git.sr.ht/~sircmpwn/aerc/commands/account"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type NextPrevMsg struct{}

func init() {
	register(NextPrevMsg{})
}

func (NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (NextPrevMsg) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (NextPrevMsg) Execute(aerc *widgets.Aerc, args []string) error {
	n, pct, err := account.ParseNextPrevMessage(args)
	if err != nil {
		return err
	}
	mv, _ := aerc.SelectedTab().(*widgets.MessageViewer)
	acct := mv.SelectedAccount()
	store := mv.Store()
	err = account.ExecuteNextPrevMessage(args, acct, pct, n)
	if err != nil {
		return err
	}
	nextMsg := store.Selected()
	if nextMsg == nil {
		aerc.RemoveTab(mv)
		return nil
	}
	lib.NewMessageStoreView(nextMsg, store, aerc.DecryptKeys,
		func(view lib.MessageView) {
			nextMv := widgets.NewMessageViewer(acct, aerc.Config(), view)
			aerc.ReplaceTab(mv, nextMv, nextMsg.Envelope.Subject)
		})
	return nil
}
