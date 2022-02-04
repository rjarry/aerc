package imap

import (
	"bufio"
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
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
	imapw.handleFetchMessages(msg, msg.Uids, items,
		func(_msg *imap.Message) error {
			reader := _msg.GetBody(section)
			textprotoHeader, err := textproto.ReadHeader(bufio.NewReader(reader))
			if err != nil {
				imapw.worker.Logger.Printf(
					"message %v: could not read header: %v", _msg.Uid, err)
				imapw.worker.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
				return nil
			}
			header := &mail.Header{message.Header{textprotoHeader}}
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info: &models.MessageInfo{
					BodyStructure: translateBodyStructure(_msg.BodyStructure),
					Envelope:      translateEnvelope(_msg.Envelope),
					Flags:         translateImapFlags(_msg.Flags),
					InternalDate:  _msg.InternalDate,
					RFC822Headers: header,
					Uid:           _msg.Uid,
				},
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleFetchMessageBodyPart(
	msg *types.FetchMessageBodyPart) {

	imapw.worker.Logger.Printf("Fetching message part")

	var partHeaderSection imap.BodySectionName
	partHeaderSection.Peek = true
	if len(msg.Part) > 0 {
		partHeaderSection.Specifier = imap.MIMESpecifier
	} else {
		partHeaderSection.Specifier = imap.HeaderSpecifier
	}
	partHeaderSection.Path = msg.Part

	var partBodySection imap.BodySectionName
	if len(msg.Part) > 0 {
		partBodySection.Specifier = imap.EntireSpecifier
	} else {
		partBodySection.Specifier = imap.TextSpecifier
	}
	partBodySection.Path = msg.Part

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchUid,
		imap.FetchBodyStructure,
		imap.FetchFlags,
		partHeaderSection.FetchItem(),
		partBodySection.FetchItem(),
	}
	imapw.handleFetchMessages(msg, []uint32{msg.Uid}, items,
		func(_msg *imap.Message) error {
			headerReader := bufio.NewReader(_msg.GetBody(&partHeaderSection))
			h, err := textproto.ReadHeader(headerReader)
			if err != nil {
				return fmt.Errorf("failed to read part header: %v", err)
			}

			part, err := message.New(message.Header{h},
				_msg.GetBody(&partBodySection))
			if err != nil {
				return fmt.Errorf("failed to create message reader: %v", err)
			}

			imapw.worker.PostMessage(&types.MessageBodyPart{
				Message: types.RespondTo(msg),
				Part: &models.MessageBodyPart{
					Reader: part.Body,
					Uid:    _msg.Uid,
				},
			}, nil)
			// Update flags (to mark message as read)
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

func (imapw *IMAPWorker) handleFetchFullMessages(
	msg *types.FetchFullMessages) {

	imapw.worker.Logger.Printf("Fetching full messages")
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchUid,
		section.FetchItem(),
	}
	imapw.handleFetchMessages(msg, msg.Uids, items,
		func(_msg *imap.Message) error {
			r := _msg.GetBody(section)
			if r == nil {
				return fmt.Errorf("could not get section %#v", section)
			}
			imapw.worker.PostMessage(&types.FullMessage{
				Message: types.RespondTo(msg),
				Content: &models.FullMessage{
					Reader: bufio.NewReader(r),
					Uid:    _msg.Uid,
				},
			}, nil)
			// Update flags (to mark message as read)
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

func (imapw *IMAPWorker) handleFetchMessages(
	msg types.WorkerMessage, uids []uint32, items []imap.FetchItem,
	procFunc func(*imap.Message) error) {

	messages := make(chan *imap.Message)
	done := make(chan error)

	go func() {
		var reterr error
		for _msg := range messages {
			imapw.seqMap[_msg.SeqNum-1] = _msg.Uid
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
	if err := imapw.client.UidFetch(set, items, messages); err != nil {
		emitErr(err)
		return
	}
	if err := <-done; err != nil {
		emitErr(err)
		return
	}
	imapw.worker.PostMessage(
		&types.Done{types.RespondTo(msg)}, nil)
}
