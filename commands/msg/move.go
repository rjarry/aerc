package msg

import (
	"errors"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Move struct{}

func init() {
	register(Move{})
}

func (_ Move) Aliases() []string {
	return []string{"mv", "move"}
}

func (_ Move) Complete(aerc *widgets.Aerc, args []string) []string {
	return commands.GetFolders(aerc, args)
}

func (_ Move) Execute(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, "p")
	if err != nil {
		return err
	}
	if optind != len(args)-1 {
		return errors.New("Usage: mv [-p] <folder>")
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
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	msg := widget.SelectedMessage()
	store := widget.Store()
	_, isMsgView := widget.(*widgets.MessageViewer)
	if isMsgView {
		aerc.RemoveTab(widget)
	}
	store.Next()
	acct.Messages().Scroll()
	store.Move([]uint32{msg.Uid}, args[optind], createParents, func(
		msg types.WorkerMessage) {

		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Messages moved.", 10*time.Second)
		case *types.Error:
			aerc.PushStatus(" "+msg.Error.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
	})
	return nil
}
