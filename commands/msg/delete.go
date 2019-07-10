package msg

import (
	"errors"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Delete struct{}

func init() {
	register(Delete{})
}

func (_ Delete) Aliases() []string {
	return []string{"delete", "delete-message"}
}

func (_ Delete) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Delete) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: :delete")
	}

	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := widget.Store()
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	_, isMsgView := widget.(*widgets.MessageViewer)
	if isMsgView {
		aerc.RemoveTab(widget)
	}
	store.Next()
	acct.Messages().Scroll()
	store.Delete([]uint32{msg.Uid}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Messages deleted.", 10*time.Second)
		case *types.Error:
			aerc.PushStatus(" "+msg.Error.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
	})
	return nil
}
