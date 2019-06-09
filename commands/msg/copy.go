package msg

import (
	"errors"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func init() {
	register("cp", Copy)
	register("copy", Copy)
}

func Copy(aerc *widgets.Aerc, args []string) error {
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
	msg := widget.SelectedMessage()
	store := widget.Store()
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
