package msg

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
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
	//caution, can be nil
	next := findNextNonDeleted(uids, store)
	store.Delete(uids, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Messages deleted.", 10*time.Second)
			mv, isMsgView := h.msgProvider.(*widgets.MessageViewer)
			if isMsgView {
				if !aerc.Config().Ui.NextMessageOnDelete {
					aerc.RemoveTab(h.msgProvider)
				} else {
					// no more messages in the list
					if next == nil {
						aerc.RemoveTab(h.msgProvider)
						acct.Messages().Select(0)
						acct.Messages().Invalidate()
						return
					}
					lib.NewMessageStoreView(next, store, aerc.Crypto, aerc.DecryptKeys,
						func(view lib.MessageView, err error) {
							if err != nil {
								aerc.PushError(err.Error())
								return
							}
							nextMv := widgets.NewMessageViewer(acct, aerc.Config(), view)
							aerc.ReplaceTab(mv, nextMv, next.Envelope.Subject)
						})
				}
			} else {
				if next == nil {
					// We deleted the last message, select the new last message
					// instead of the first message
					acct.Messages().Select(0)
				}
			}
			acct.Messages().Invalidate()
		case *types.Error:
			aerc.PushError(msg.Error.Error())
		case *types.Unsupported:
			// notmuch doesn't support it, we want the user to know
			aerc.PushError(" error, unsupported for this worker")
		}
	})
	return nil
}

func findNextNonDeleted(deleted []uint32, store *lib.MessageStore) *models.MessageInfo {
	var next, previous *models.MessageInfo
	stepper := []func(){store.Next, store.Prev}
	for _, stepFn := range stepper {
		for {
			next = store.Selected()
			if next != nil && !contains(deleted, next.Uid) {
				return next
			}
			if next == nil || previous == next {
				break
			}
			stepFn()
			previous = next
		}
	}

	if next != nil {
		store.Select(next.Uid)
	}
	return next
}

func contains(uids []uint32, uid uint32) bool {
	for _, item := range uids {
		if item == uid {
			return true
		}
	}
	return false
}
