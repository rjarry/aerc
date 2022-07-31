package msg

import (
	"errors"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
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
	var createParents bool
	for _, opt := range opts {
		if opt.Option == 'p' {
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
	_, isMsgView := h.msgProvider.(*widgets.MessageViewer)
	if isMsgView {
		aerc.RemoveTab(h.msgProvider)
	}
	store.ClearVisualMark()
	findNextNonDeleted(uids, store)
	joinedArgs := strings.Join(args[optind:], " ")
	store.Move(uids, joinedArgs, createParents, func(
		msg types.WorkerMessage,
	) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Message moved to "+joinedArgs, 10*time.Second)
		case *types.Error:
			store.Remark()
			aerc.PushError(msg.Error.Error())
		}
	})
	return nil
}
