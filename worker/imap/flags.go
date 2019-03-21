package imap

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func (imapw *IMAPWorker) handleDeleteMessages(msg *types.DeleteMessages) {
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := imapw.client.UidStore(&msg.Uids, item, flags, nil); err != nil {
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
