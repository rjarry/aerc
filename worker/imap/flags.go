package imap

import (
	"fmt"

	"github.com/emersion/go-imap"

	"git.sr.ht/~rjarry/aerc/logging"
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
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.AnsweredFlag}
	if !msg.Answered {
		item = imap.FormatFlagsOp(imap.RemoveFlags, true)
		flags = []interface{}{imap.AnsweredFlag}
	}
	uids := toSeqSet(msg.Uids)
	emitErr := func(err error) {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	}
	if err := imapw.client.UidStore(uids, item, flags, nil); err != nil {
		emitErr(err)
		return
	}
	imapw.worker.PostAction(&types.FetchMessageHeaders{
		Uids: msg.Uids,
	}, func(_msg types.WorkerMessage) {
		switch m := _msg.(type) {
		case *types.Error:
			err := fmt.Errorf("handleAnsweredMessages: %w", m.Error)
			logging.Errorf("could not fetch headers: %v", err)
			emitErr(err)
		case *types.Done:
			imapw.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
		}
	})
}

func (imapw *IMAPWorker) handleFlagMessages(msg *types.FlagMessages) {
	flags := []interface{}{flagToImap[msg.Flag]}
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	if !msg.Enable {
		item = imap.FormatFlagsOp(imap.RemoveFlags, true)
	}
	uids := toSeqSet(msg.Uids)
	emitErr := func(err error) {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	}
	if err := imapw.client.UidStore(uids, item, flags, nil); err != nil {
		emitErr(err)
		return
	}
	imapw.worker.PostAction(&types.FetchMessageHeaders{
		Uids: msg.Uids,
	}, func(_msg types.WorkerMessage) {
		switch m := _msg.(type) {
		case *types.Error:
			err := fmt.Errorf("handleFlagMessages: %w", m.Error)
			logging.Errorf("could not fetch headers: %v", err)
			emitErr(err)
		case *types.Done:
			imapw.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
		}
	})
}
