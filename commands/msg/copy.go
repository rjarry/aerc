package msg

import (
	"errors"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Copy struct{}

func init() {
	register(Copy{})
}

func (_ Copy) Aliases() []string {
	return []string{"copy"}
}

func (_ Copy) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Copy) Execute(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, "p")
	if err != nil {
		return err
	}
	if optind != len(args)-1 {
		return errors.New("Usage: cp [-p] <folder>")
	}
	var (
		createParents bool
	)
	for _, opt := range opts {
		switch opt.Option {
		case 'p':
			createParents = true
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
	store.Copy([]uint32{msg.Uid}, args[optind], createParents, func(
		msg types.WorkerMessage) {

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
