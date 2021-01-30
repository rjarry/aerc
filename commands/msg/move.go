package msg

import (
	"errors"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Move struct{}

func init() {
	register(Move{})
}

func (Move) Aliases() []string {
	return []string{"mv", "move"}
}

func (Move) Complete(aerc *widgets.Aerc, args []string) []string {
	return commands.GetFolders(aerc, args)
}

func (Move) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		return errors.New("Usage: mv [-p] <folder>")
	}
	opts, optind, err := getopt.Getopts(args, "p")
	if err != nil {
		return err
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

	h := newHelper(aerc)
	store, err := h.store()
	if err != nil {
		return err
	}
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}
	acct, err := h.account()
	if err != nil {
		return err
	}
	_, isMsgView := h.msgProvider.(*widgets.MessageViewer)
	if isMsgView {
		aerc.RemoveTab(h.msgProvider)
	}
	store.Next()
	acct.Messages().Invalidate()
	joinedArgs := strings.Join(args[optind:], " ")
	store.Move(uids, joinedArgs, createParents, func(
		msg types.WorkerMessage) {

		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Message moved to "+joinedArgs, 10*time.Second)
		case *types.Error:
			aerc.PushError(msg.Error.Error())
		}
	})
	return nil
}
