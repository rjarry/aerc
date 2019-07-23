package msg

import (
	"errors"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/models"
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
	opts, optind, err := getopt.Getopts(args, "t")
	if err != nil {
		return err
	}
	if optind != len(args) {
		return errors.New("Usage: " + args[0] + " [-t]")
	}
	var toggle bool

	for _, opt := range opts {
		switch opt.Option {
		case 't':
			toggle = true
		}
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
	newReadState := true
	if toggle {
		newReadState = true
		for _, flag := range msg.Flags {
			if flag == models.SeenFlag {
				newReadState = false
			}
		}
	} else if args[0] == "read" {
		newReadState = true
	}
	store.Read([]uint32{msg.Uid}, newReadState, func(
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
