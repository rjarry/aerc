package imap

import (
	"bufio"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"

	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func (imapw *IMAPWorker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders) {

	imapw.worker.Logger.Printf("Fetching message headers")
	section := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			Specifier: imap.HeaderSpecifier,
		},
		Peek: true,
	}

	items := []imap.FetchItem{
		imap.FetchBodyStructure,
		imap.FetchEnvelope,
		imap.FetchInternalDate,
		imap.FetchFlags,
		imap.FetchUid,
		section.FetchItem(),
	}

	imapw.handleFetchMessages(msg, &msg.Uids, items, section)
}

func (imapw *IMAPWorker) handleFetchMessageBodyPart(
	msg *types.FetchMessageBodyPart) {

	imapw.worker.Logger.Printf("Fetching message part")
	section := &imap.BodySectionName{}
	section.Path = msg.Part
	items := []imap.FetchItem{section.FetchItem()}
	uids := imap.SeqSet{}
	uids.AddNum(msg.Uid)
	imapw.handleFetchMessages(msg, &uids, items, section)
}

func (imapw *IMAPWorker) handleFetchFullMessages(
	msg *types.FetchFullMessages) {

	imapw.worker.Logger.Printf("Fetching full messages")
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}
	imapw.handleFetchMessages(msg, &msg.Uids, items, section)
}

func (imapw *IMAPWorker) handleFetchMessages(
	msg types.WorkerMessage, uids *imap.SeqSet, items []imap.FetchItem,
	section *imap.BodySectionName) {

	messages := make(chan *imap.Message)
	done := make(chan interface{})

	go func() {
		for _msg := range messages {
			imapw.seqMap[_msg.SeqNum-1] = _msg.Uid
			switch msg.(type) {
			case *types.FetchMessageHeaders:
				reader := _msg.GetBody(section)
				textprotoHeader, err := textproto.ReadHeader(bufio.NewReader(reader))
				var header *mail.Header
				if err == nil {
					header = &mail.Header{message.Header{textprotoHeader}}
				}
				imapw.worker.PostMessage(&types.MessageInfo{
					Message:       types.RespondTo(msg),
					BodyStructure: _msg.BodyStructure,
					Envelope:      _msg.Envelope,
					Flags:         _msg.Flags,
					InternalDate:  _msg.InternalDate,
					RFC822Headers: header,
					Uid:           _msg.Uid,
				}, nil)
			case *types.FetchFullMessages:
				reader := _msg.GetBody(section)
				imapw.worker.PostMessage(&types.FullMessage{
					Message: types.RespondTo(msg),
					Reader:  reader,
					Uid:     _msg.Uid,
				}, nil)
			case *types.FetchMessageBodyPart:
				reader := _msg.GetBody(section)
				imapw.worker.PostMessage(&types.MessageBodyPart{
					Message: types.RespondTo(msg),
					Reader:  reader,
					Uid:     _msg.Uid,
				}, nil)
			}
		}
		done <- nil
	}()

	if err := imapw.client.UidFetch(uids, items, messages); err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		<-done
		imapw.worker.PostMessage(
			&types.Done{types.RespondTo(msg)}, nil)
	}
}
