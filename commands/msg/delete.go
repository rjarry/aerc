package msg

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Delete struct {
	MultiFileStrategy *types.MultiFileStrategy `opt:"-m" action:"ParseMFS" complete:"CompleteMFS"`
}

func init() {
	commands.Register(Delete{})
}

func (Delete) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (Delete) Aliases() []string {
	return []string{"delete", "delete-message"}
}

func (d *Delete) ParseMFS(arg string) error {
	if arg != "" {
		mfs, ok := types.StrToStrategy[arg]
		if !ok {
			return fmt.Errorf("invalid multi-file strategy %s", arg)
		}
		d.MultiFileStrategy = &mfs
	}
	return nil
}

func (Delete) CompleteMFS(arg string) []string {
	return commands.FilterList(types.StrategyStrs(), arg, nil)
}

func (d Delete) Execute(args []string) error {
	h := newHelper()
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
	sel := store.Selected()
	marker := store.Marker()
	marker.ClearVisualMark()
	// caution, can be nil
	next := findNextNonDeleted(uids, store)
	store.Delete(uids, d.MultiFileStrategy, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			var s string
			if len(uids) > 1 {
				s = "%d messages deleted"
			} else {
				s = "%d message deleted"
			}
			app.PushStatus(fmt.Sprintf(s, len(uids)), 10*time.Second)
			mv, isMsgView := h.msgProvider.(*app.MessageViewer)
			if isMsgView {
				if !config.Ui.NextMessageOnDelete {
					app.RemoveTab(h.msgProvider, true)
				} else {
					// no more messages in the list
					if next == nil {
						app.RemoveTab(h.msgProvider, true)
						acct.Messages().Select(-1)
						ui.Invalidate()
						return
					}
					lib.NewMessageStoreView(next, mv.MessageView().SeenFlagSet(),
						store, app.CryptoProvider(), app.DecryptKeys,
						func(view lib.MessageView, err error) {
							if err != nil {
								app.PushError(err.Error())
								return
							}
							nextMv := app.NewMessageViewer(acct, view)
							app.ReplaceTab(mv, nextMv, next.Envelope.Subject, true)
						})
				}
			} else {
				if next == nil {
					// We deleted the last message, select the new last message
					// instead of the first message
					acct.Messages().Select(-1)
				}
			}
		case *types.Error:
			marker.Remark()
			store.Select(sel.Uid)
			app.PushError(msg.Error.Error())
		case *types.Unsupported:
			marker.Remark()
			store.Select(sel.Uid)
			// notmuch doesn't support it, we want the user to know
			app.PushError(" error, unsupported for this worker")
		}
	})
	return nil
}

func findNextNonDeleted(deleted []uint32, store *lib.MessageStore) *models.MessageInfo {
	var next, previous *models.MessageInfo
	stepper := []func(){store.Next, store.Prev}
	for _, stepFn := range stepper {
		previous = nil
		for {
			next = store.Selected()
			if next != nil && !contains(deleted, next.Uid) {
				if _, deleted := store.Deleted[next.Uid]; !deleted {
					return next
				}
			}
			if next == nil || previous == next {
				// If previous == next, this is the last
				// message. Set next to nil either way
				next = nil
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
