package msg

import (
	"errors"
	"time"

	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Delete struct{}

func init() {
	register(Delete{})
}

func (Delete) Aliases() []string {
	return []string{"delete", "delete-message"}
}

func (Delete) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Delete) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: :delete")
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
	store.Delete(uids, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Messages deleted.", 10*time.Second)
		case *types.Error:
			aerc.PushError(" " + msg.Error.Error())
		}
	})

	//caution, can be nil
	next := findNextNonDeleted(uids, store)

	mv, isMsgView := h.msgProvider.(*widgets.MessageViewer)
	if isMsgView {
		if !aerc.Config().Ui.NextMessageOnDelete {
			aerc.RemoveTab(h.msgProvider)
		} else {
			// no more messages in the list
			if next == nil {
				aerc.RemoveTab(h.msgProvider)
				acct.Messages().Scroll()
				return nil
			}
			lib.NewMessageStoreView(next, store, aerc.DecryptKeys,
				func(view lib.MessageView, err error) {
					if err != nil {
						aerc.PushError(err.Error())
						return
					}
					nextMv := widgets.NewMessageViewer(acct, aerc.Config(), view)
					aerc.ReplaceTab(mv, nextMv, next.Envelope.Subject)
				})
		}
	}
	acct.Messages().Scroll()
	return nil
}

func findNextNonDeleted(deleted []uint32, store *lib.MessageStore) *models.MessageInfo {
	selected := store.Selected()
	if !contains(deleted, selected.Uid) {
		return selected
	}
	for {
		store.Next()
		next := store.Selected()
		if next == selected || next == nil {
			// the last message is in the deleted state or doesn't exist
			return nil
		}
		return next
	}
	return nil // Never reached
}

func contains(uids []uint32, uid uint32) bool {
	for _, item := range uids {
		if item == uid {
			return true
		}
	}
	return false
}
