package imap

import (
	"fmt"
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc/worker/types"
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
	var deleted []uint32
	ch := make(chan uint32)
	done := make(chan interface{})
	go func() {
		for seqNum := range ch {
			i := seqNum - 1
			deleted = append(deleted, imapw.seqMap[i])
			imapw.seqMap = append(imapw.seqMap[:i], imapw.seqMap[i+1:]...)
		}
		done <- nil
	}()
	if err := imapw.client.Expunge(ch); err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		<-done
		imapw.worker.PostMessage(&types.MessagesDeleted{
			Message: types.RespondTo(msg),
			Uids:    deleted,
		}, nil)
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
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
			err := fmt.Errorf("handleAnsweredMessages: %v", m.Error)
			imapw.worker.Logger.Printf("could not fetch headers: %s", err)
			emitErr(err)
		case *types.Done:
			imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
		}
	})
}

func (imapw *IMAPWorker) handleReadMessages(msg *types.ReadMessages) {
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.SeenFlag}
	if !msg.Read {
		item = imap.FormatFlagsOp(imap.RemoveFlags, true)
		flags = []interface{}{imap.SeenFlag}
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
			err := fmt.Errorf("handleReadMessages: %v", m.Error)
			imapw.worker.Logger.Printf("could not fetch headers: %s", err)
			emitErr(err)
		case *types.Done:
			imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
		}
	})
}
