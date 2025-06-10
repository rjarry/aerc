package imap

import (
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

// drainUpdates will drain the updates channel. For some operations, the imap
// server will send unilateral messages. If they arrive while another operation
// is in progress, the buffered updates channel can fill up and cause a freeze
// of the entire backend. Avoid this by draining the updates channel and only
// process the Message and Expunge updates.
//
// To stop the draining, close the returned struct.
func (imapw *IMAPWorker) drainUpdates() *drainCloser {
	done := make(chan struct{})
	go func() {
		defer log.PanicHandler()
		for {
			select {
			case update := <-imapw.updates:
				switch update.(type) {
				case *client.MessageUpdate,
					*client.ExpungeUpdate:
					imapw.handleImapUpdate(update)
				}
			case <-done:
				return
			}
		}
	}()
	return &drainCloser{done}
}

type drainCloser struct {
	done chan struct{}
}

func (d *drainCloser) Close() error {
	close(d.done)
	return nil
}

func (imapw *IMAPWorker) handleDeleteMessages(msg *types.DeleteMessages) {
	drain := imapw.drainUpdates()
	defer drain.Close()

	// Build provider-dependent EXPUNGE handler.
	imapw.BuildExpungeHandler(models.UidToUint32List(msg.Uids))

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []any{imap.DeletedFlag}
	uids := toSeqSet(msg.Uids)
	if err := imapw.client.UidStore(uids, item, flags, nil); err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
		return
	}
	if err := imapw.client.Expunge(nil); err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
	}
}

func (imapw *IMAPWorker) handleAnsweredMessages(msg *types.AnsweredMessages) {
	item := imap.FormatFlagsOp(imap.AddFlags, false)
	flags := []any{imap.AnsweredFlag}
	if !msg.Answered {
		item = imap.FormatFlagsOp(imap.RemoveFlags, false)
	}
	imapw.handleStoreOps(msg, msg.Uids, item, flags,
		func(_msg *imap.Message) error {
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info: &models.MessageInfo{
					Flags: translateImapFlags(_msg.Flags),
					Uid:   models.Uint32ToUid(_msg.Uid),
				},
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleFlagMessages(msg *types.FlagMessages) {
	flags := []any{flagToImap[msg.Flags]}
	item := imap.FormatFlagsOp(imap.AddFlags, false)
	if !msg.Enable {
		item = imap.FormatFlagsOp(imap.RemoveFlags, false)
	}
	imapw.handleStoreOps(msg, msg.Uids, item, flags,
		func(_msg *imap.Message) error {
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info: &models.MessageInfo{
					Flags: translateImapFlags(_msg.Flags),
					Uid:   models.Uint32ToUid(_msg.Uid),
				},
				ReplaceFlags: true,
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleStoreOps(
	msg types.WorkerMessage, uids []models.UID, item imap.StoreItem, flag any,
	procFunc func(*imap.Message) error,
) {
	messages := make(chan *imap.Message)
	done := make(chan error)

	go func() {
		defer log.PanicHandler()

		var reterr error
		for _msg := range messages {
			err := procFunc(_msg)
			if err != nil {
				if reterr == nil {
					reterr = err
				}
				// drain the channel upon error
				for range messages {
				}
			}
		}
		done <- reterr
	}()

	emitErr := func(err error) {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	}

	set := toSeqSet(uids)
	if err := imapw.client.UidStore(set, item, flag, messages); err != nil {
		emitErr(err)
		return
	}
	if err := <-done; err != nil {
		emitErr(err)
		return
	}
	imapw.worker.PostAction(&types.CheckMail{
		Directories: []string{imapw.selected.Name},
	}, nil)
	imapw.worker.PostMessage(
		&types.Done{Message: types.RespondTo(msg)}, nil)
}
