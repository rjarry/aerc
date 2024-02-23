package msg

import (
	"bytes"
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/marker"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Move struct {
	CreateFolders     bool                     `opt:"-p"`
	Account           string                   `opt:"-a" complete:"CompleteAccount"`
	MultiFileStrategy *types.MultiFileStrategy `opt:"-m" action:"ParseMFS" complete:"CompleteMFS"`
	Folder            string                   `opt:"folder" complete:"CompleteFolder"`
}

func init() {
	commands.Register(Move{})
}

func (Move) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (Move) Aliases() []string {
	return []string{"mv", "move"}
}

func (m *Move) ParseMFS(arg string) error {
	if arg != "" {
		mfs, ok := types.StrToStrategy[arg]
		if !ok {
			return fmt.Errorf("invalid multi-file strategy %s", arg)
		}
		m.MultiFileStrategy = &mfs
	}
	return nil
}

func (*Move) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, commands.QuoteSpace)
}

func (m *Move) CompleteFolder(arg string) []string {
	var acct *app.AccountView
	if len(m.Account) > 0 {
		acct, _ = app.Account(m.Account)
	} else {
		acct = app.SelectedAccount()
	}
	if acct == nil {
		return nil
	}
	return commands.FilterList(acct.Directories().List(), arg, nil)
}

func (Move) CompleteMFS(arg string) []string {
	return commands.FilterList(types.StrategyStrs(), arg, nil)
}

func (m Move) Execute(args []string) error {
	h := newHelper()
	acct, err := h.account()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}

	next := findNextNonDeleted(uids, store)
	marker := store.Marker()
	marker.ClearVisualMark()

	if len(m.Account) == 0 {
		store.Move(uids, m.Folder, m.CreateFolders, m.MultiFileStrategy,
			func(msg types.WorkerMessage) {
				m.CallBack(msg, acct, uids, next, marker, false)
			})
		return nil
	}

	destAcct, err := app.Account(m.Account)
	if err != nil {
		return err
	}

	destStore := destAcct.Store()
	if destStore == nil {
		app.PushError(fmt.Sprintf("No message store in %s", m.Account))
		return nil
	}

	var messages []*types.FullMessage
	fetchDone := make(chan bool, 1)
	store.FetchFull(uids, func(fm *types.FullMessage) {
		messages = append(messages, fm)
		if len(messages) == len(uids) {
			fetchDone <- true
		}
	})

	// Since this operation can take some time with some backends
	// (e.g. IMAP), provide some feedback to inform the user that
	// something is happening
	app.PushStatus("Moving messages...", 10*time.Second)

	var appended []uint32
	var timeout bool
	go func() {
		defer log.PanicHandler()

		select {
		case <-fetchDone:
			break
		case <-time.After(30 * time.Second):
			// TODO: find a better way to determine if store.FetchFull()
			// has finished with some errors.
			app.PushError("Failed to fetch all messages")
			if len(messages) == 0 {
				return
			}
		}

	AppendLoop:
		for _, fm := range messages {
			done := make(chan bool, 1)
			uid := fm.Content.Uid
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(fm.Content.Reader)
			if err != nil {
				log.Errorf("could not get reader for uid %d", uid)
				break
			}
			destStore.Append(
				m.Folder,
				models.SeenFlag,
				time.Now(),
				buf,
				buf.Len(),
				func(msg types.WorkerMessage) {
					switch msg := msg.(type) {
					case *types.Done:
						appended = append(appended, uid)
						done <- true
					case *types.Error:
						log.Errorf("AppendMessage failed: %v", msg.Error)
						done <- false
					}
				},
			)
			select {
			case ok := <-done:
				if !ok {
					break AppendLoop
				}
			case <-time.After(30 * time.Second):
				log.Warnf("timed-out: appended %d of %d", len(appended), len(messages))
				timeout = true
				break AppendLoop
			}
		}
		if len(appended) > 0 {
			mfs := types.Refuse
			store.Delete(appended, &mfs, func(msg types.WorkerMessage) {
				m.CallBack(msg, acct, appended, next, marker, timeout)
			})
		}
	}()
	return nil
}

func (m Move) CallBack(
	msg types.WorkerMessage,
	acct *app.AccountView,
	uids []uint32,
	next *models.MessageInfo,
	marker marker.Marker,
	timeout bool,
) {
	store := acct.Store()
	sel := store.Selected()

	dest := m.Folder
	if len(m.Account) > 0 {
		dest = fmt.Sprintf("%s in %s", m.Folder, m.Account)
	}

	switch msg := msg.(type) {
	case *types.Done:
		var s string
		if len(uids) > 1 {
			s = "%d messages moved to %s"
		} else {
			s = "%d message moved to %s"
		}
		if timeout {
			s = "timed-out: only " + s
			app.PushError(fmt.Sprintf(s, len(uids), dest))
		} else {
			app.PushStatus(fmt.Sprintf(s, len(uids), dest), 10*time.Second)
		}
		handleDone(acct, next, store)
	case *types.Error:
		app.PushError(msg.Error.Error())
		marker.Remark()
	case *types.Unsupported:
		marker.Remark()
		store.Select(sel.Uid)
		app.PushError("error, unsupported for this worker")
	}
}

func handleDone(
	acct *app.AccountView,
	next *models.MessageInfo,
	store *lib.MessageStore,
) {
	h := newHelper()
	mv, isMsgView := h.msgProvider.(*app.MessageViewer)
	switch {
	case isMsgView && !config.Ui.NextMessageOnDelete:
		app.RemoveTab(h.msgProvider, true)
	case isMsgView:
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
	default:
		if next == nil {
			// We moved the last message, select the new last message
			// instead of the first message
			acct.Messages().Select(-1)
		}
	}
}
