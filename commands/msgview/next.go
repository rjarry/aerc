package msgview

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
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
	mv, _ := aerc.SelectedTabContent().(*widgets.MessageViewer)
	acct := mv.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := mv.Store()
	if store == nil {
		return fmt.Errorf("Cannot perform action. No message store set.")
	}
	err = account.ExecuteNextPrevMessage(args, acct, pct, n)
	if err != nil {
		return err
	}
	executeNextPrev := func(nextMsg *models.MessageInfo) {
		lib.NewMessageStoreView(nextMsg, mv.MessageView().SeenFlagSet(),
			store, aerc.Crypto, aerc.DecryptKeys,
			func(view lib.MessageView, err error) {
				if err != nil {
					aerc.PushError(err.Error())
					return
				}
				nextMv := widgets.NewMessageViewer(acct, view)
				aerc.ReplaceTab(mv, nextMv,
					nextMsg.Envelope.Subject, true)
			})
	}
	if nextMsg := store.Selected(); nextMsg != nil {
		executeNextPrev(nextMsg)
	} else {
		store.FetchHeaders([]uint32{store.SelectedUid()},
			func(msg types.WorkerMessage) {
				if m, ok := msg.(*types.MessageInfo); ok {
					executeNextPrev(m.Info)
				}
			})
	}

	return nil
}
