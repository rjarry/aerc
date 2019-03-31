package imap

import (
	"github.com/emersion/go-imap"
	"github.com/mohamedattahri/mail"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func (imapw *IMAPWorker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders) {

	imapw.worker.Logger.Printf("Fetching message headers")
	items := []imap.FetchItem{
		imap.FetchBodyStructure,
		imap.FetchEnvelope,
		imap.FetchInternalDate,
		imap.FetchFlags,
		imap.FetchUid,
	}

	imapw.handleFetchMessages(msg, &msg.Uids, items)
}

func (imapw *IMAPWorker) handleFetchMessageBodies(
	msg *types.FetchMessageBodies) {

	imapw.worker.Logger.Printf("Fetching message bodies")
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}
	imapw.handleFetchMessages(msg, &msg.Uids, items)
}

func (imapw *IMAPWorker) handleFetchMessages(
	msg types.WorkerMessage, uids *imap.SeqSet, items []imap.FetchItem) {

	go func() {
		messages := make(chan *imap.Message)
		done := make(chan error, 1)
		go func() {
			done <- imapw.client.UidFetch(uids, items, messages)
		}()
		go func() {
			section := &imap.BodySectionName{}
			for _msg := range messages {
				imapw.seqMap[_msg.SeqNum-1] = _msg.Uid
				if reader := _msg.GetBody(section); reader != nil {
					email, err := mail.ReadMessage(reader)
					if err != nil {
						imapw.worker.PostMessage(&types.Error{
							Message: types.RespondTo(msg),
							Error:   err,
						}, nil)
					}
					imapw.worker.PostMessage(&types.MessageBody{
						Mail: email,
						Uid:  _msg.Uid,
					}, nil)
				} else {
					imapw.worker.PostMessage(&types.MessageInfo{
						BodyStructure: _msg.BodyStructure,
						Envelope:      _msg.Envelope,
						Flags:         _msg.Flags,
						InternalDate:  _msg.InternalDate,
						Uid:           _msg.Uid,
					}, nil)
				}
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
