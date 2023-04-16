package imap

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleDeleteMessages(msg *types.DeleteMessages) {
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
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
	flags := []interface{}{imap.AnsweredFlag}
	if !msg.Answered {
		item = imap.FormatFlagsOp(imap.RemoveFlags, false)
	}
	imapw.handleStoreOps(msg, msg.Uids, item, flags,
		func(_msg *imap.Message) error {
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info: &models.MessageInfo{
					Flags: translateImapFlags(_msg.Flags),
					Uid:   _msg.Uid,
				},
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleFlagMessages(msg *types.FlagMessages) {
	flags := []interface{}{flagToImap[msg.Flags]}
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
					Uid:   _msg.Uid,
				},
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleStoreOps(
	msg types.WorkerMessage, uids []uint32, item imap.StoreItem, flag interface{},
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
