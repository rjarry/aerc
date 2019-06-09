package account

import (
	"errors"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func init() {
	register("mkdir", Mkdir)
}

func Mkdir(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: :mkdir <name>")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	name := args[1]
	acct.Worker().PostAction(&types.CreateDirectory{
		Directory: name,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Directory created.", 10*time.Second)
			acct.Directories().Select(name)
		case *types.Error:
			aerc.PushStatus(" "+msg.Error.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
	})
	return nil
}
