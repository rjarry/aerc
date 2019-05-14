package account

import (
	"errors"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func init() {
	register("cp", Copy)
	register("copy", Copy)
}

func Copy(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: mv <folder>")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	msg := acct.Messages().Selected()
	store := acct.Messages().Store()
	store.Copy([]uint32{msg.Uid}, args[1], func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Messages copied.", 10*time.Second)
		case *types.Error:
			aerc.PushStatus(" "+msg.Error.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
	})
	return nil
}
