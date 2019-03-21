package imap

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func (imapw *IMAPWorker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders) {

	imapw.worker.Logger.Printf("Fetching message headers")

	go func() {
		messages := make(chan *imap.Message)
		done := make(chan error, 1)
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchInternalDate,
			imap.FetchFlags,
			imap.FetchUid,
		}
		go func() {
			done <- imapw.client.UidFetch(&msg.Uids, items, messages)
		}()
		go func() {
			for msg := range messages {
				imapw.seqMap[msg.SeqNum-1] = msg.Uid
				imapw.worker.PostMessage(&types.MessageInfo{
					Envelope:     msg.Envelope,
					Flags:        msg.Flags,
					InternalDate: msg.InternalDate,
					Uid:          msg.Uid,
				}, nil)
			}
			if err := <-done; err != nil {
				imapw.worker.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
			} else {
				imapw.worker.PostMessage(
					&types.Done{types.RespondTo(msg)}, nil)
			}
		}()
	}()
}
