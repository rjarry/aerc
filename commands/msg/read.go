package msg

import (
	"errors"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func init() {
	register("read", Read)
	register("unread", Read)
}

func Read(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: " + args[0])
	}

	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	msg := widget.SelectedMessage()
	store := widget.Store()
	store.Read([]uint32{msg.Uid}, args[0] == "read", func(
		msg types.WorkerMessage) {

		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Messages updated.", 10*time.Second)
		case *types.Error:
			aerc.PushStatus(" "+msg.Error.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
	})
	return nil
}
