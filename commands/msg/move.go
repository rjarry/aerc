package msg

import (
	"errors"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~sircmpwn/getopt"
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
	acct, err := h.account()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	msgs, err := h.messages()
	if err != nil {
		return err
	}
	var uids []uint32
	for _, msg := range msgs {
		uids = append(uids, msg.Uid)
	}
	marker := store.Marker()
	marker.ClearVisualMark()
	next := findNextNonDeleted(uids, store)
	joinedArgs := strings.Join(args[optind:], " ")

	store.Move(uids, joinedArgs, createParents, func(
		msg types.WorkerMessage,
	) {
		switch msg := msg.(type) {
		case *types.Done:
			handleDone(aerc, acct, next, "Messages moved to "+joinedArgs, store)
		case *types.Error:
			aerc.PushError(msg.Error.Error())
			marker.Remark()
		}
	})

	return nil
}

func handleDone(
	aerc *widgets.Aerc,
	acct *widgets.AccountView,
	next *models.MessageInfo,
	message string,
	store *lib.MessageStore,
) {
	h := newHelper(aerc)
	aerc.PushStatus(message, 10*time.Second)
	mv, isMsgView := h.msgProvider.(*widgets.MessageViewer)
	switch {
	case isMsgView && !config.Ui.NextMessageOnDelete:
		aerc.RemoveTab(h.msgProvider)
	case isMsgView:
		if next == nil {
			aerc.RemoveTab(h.msgProvider)
			acct.Messages().Select(-1)
			ui.Invalidate()
			return
		}
		lib.NewMessageStoreView(next, mv.MessageView().SeenFlagSet(),
			store, aerc.Crypto, aerc.DecryptKeys,
			func(view lib.MessageView, err error) {
				if err != nil {
					aerc.PushError(err.Error())
					return
				}
				nextMv := widgets.NewMessageViewer(acct, view)
				aerc.ReplaceTab(mv, nextMv, next.Envelope.Subject)
			})
	default:
		if next == nil {
			// We moved the last message, select the new last message
			// instead of the first message
			acct.Messages().Select(-1)
		}
	}
}
