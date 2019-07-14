package msg

import (
	"errors"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Read struct{}

func init() {
	register(Read{})
}

func (_ Read) Aliases() []string {
	return []string{"read", "unread"}
}

func (_ Read) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Read) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: " + args[0])
	}

	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
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
