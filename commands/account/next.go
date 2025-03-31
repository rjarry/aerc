package account

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type NextPrevMsg struct {
	Amount  int `opt:"n" minus:"true" default:"1" metavar:"<n>[%]" action:"ParseAmount"`
	Percent bool
}

func init() {
	commands.Register(NextPrevMsg{})
}

func (NextPrevMsg) Description() string {
	return "Select the next or previous message in the message list."
}

func (NextPrevMsg) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
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

func (np NextPrevMsg) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return fmt.Errorf("No message store set.")
	}

	n := np.Amount
	if np.Percent {
		n = int(float64(acct.Messages().Height()) * (float64(n) / 100.0))
	}
	if args[0] == "prev-message" || args[0] == "prev" {
		store.NextPrev(-n)
	} else {
		store.NextPrev(n)
	}

	if mv, ok := app.SelectedTabContent().(*app.MessageViewer); ok {
		reloadViewer := func(nextMsg *models.MessageInfo) {
			if nextMsg.Error != nil {
				app.PushError(nextMsg.Error.Error())
				return
			}
			lib.NewMessageStoreView(nextMsg, mv.MessageView().SeenFlagSet(),
				store, app.CryptoProvider(), app.DecryptKeys,
				func(view lib.MessageView, err error) {
					if err != nil {
						app.PushError(err.Error())
						return
					}
					nextMv, err := app.NewMessageViewer(acct, view)
					if err != nil {
						app.PushError(err.Error())
						return
					}
					app.ReplaceTab(mv, nextMv,
						nextMsg.Envelope.Subject, true)
				})
		}
		if nextMsg := store.Selected(); nextMsg != nil {
			reloadViewer(nextMsg)
		} else {
			store.FetchHeaders([]models.UID{store.SelectedUid()},
				func(msg types.WorkerMessage) {
					if m, ok := msg.(*types.MessageInfo); ok {
						reloadViewer(m.Info)
					}
				})
		}
	}

	ui.Invalidate()

	return nil
}
