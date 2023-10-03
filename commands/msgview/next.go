package msgview

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type NextPrevMsg struct {
	Amount  int `opt:"n" default:"1" metavar:"N[%]" action:"ParseAmount"`
	Percent bool
}

func init() {
	register(NextPrevMsg{})
}

func (np *NextPrevMsg) ParseAmount(arg string) error {
	if strings.HasSuffix(arg, "%") {
		np.Percent = true
		arg = strings.TrimSuffix(arg, "%")
	}
	i, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return err
	}
	np.Amount = int(i)
	return nil
}

func (NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (NextPrevMsg) Complete(args []string) []string {
	return nil
}

func (np NextPrevMsg) Execute(args []string) error {
	cmd := account.NextPrevMsg{Amount: np.Amount, Percent: np.Percent}
	err := cmd.Execute(args)
	if err != nil {
		return err
	}

	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
	acct := mv.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := mv.Store()
	if store == nil {
		return fmt.Errorf("Cannot perform action. No message store set.")
	}
	executeNextPrev := func(nextMsg *models.MessageInfo) {
		lib.NewMessageStoreView(nextMsg, mv.MessageView().SeenFlagSet(),
			store, app.CryptoProvider(), app.DecryptKeys,
			func(view lib.MessageView, err error) {
				if err != nil {
					app.PushError(err.Error())
					return
				}
				nextMv := app.NewMessageViewer(acct, view)
				app.ReplaceTab(mv, nextMv,
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
