package imap

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"

	"git.sr.ht/~sircmpwn/aerc/models"
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
	imapw.handleFetchMessages(msg, msg.Uids, items, section)
}

func (imapw *IMAPWorker) handleFetchMessageBodyPart(
	msg *types.FetchMessageBodyPart) {

	imapw.worker.Logger.Printf("Fetching message part")
	section := &imap.BodySectionName{}
	section.Path = msg.Part
	items := []imap.FetchItem{
		imap.FetchFlags,
		imap.FetchUid,
		section.FetchItem(),
	}
	imapw.handleFetchMessages(msg, []uint32{msg.Uid}, items, section)
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
	imapw.handleFetchMessages(msg, msg.Uids, items, section)
}

func (imapw *IMAPWorker) handleFetchMessages(
	msg types.WorkerMessage, uids []uint32, items []imap.FetchItem,
	section *imap.BodySectionName) {

	messages := make(chan *imap.Message)
	done := make(chan error)

	go func() {
		for _msg := range messages {
			imapw.seqMap[_msg.SeqNum-1] = _msg.Uid
			switch msg := msg.(type) {
			case *types.FetchMessageHeaders:
				reader := _msg.GetBody(section)
				textprotoHeader, err := textproto.ReadHeader(bufio.NewReader(reader))
				var header *mail.Header
				if err == nil {
					header = &mail.Header{message.Header{textprotoHeader}}
				}
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
			case *types.FetchFullMessages:
				r := _msg.GetBody(section)
				if r == nil {
					done <- fmt.Errorf("could not get section %#v", section)
					return
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
			case *types.FetchMessageBodyPart:
				reader, err := getDecodedPart(msg, _msg, section)
				if err != nil {
					done <- err
					return
				}
				imapw.worker.PostMessage(&types.MessageBodyPart{
					Message: types.RespondTo(msg),
					Part: &models.MessageBodyPart{
						Reader: reader,
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
			}
		}
		done <- nil
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

func getDecodedPart(task *types.FetchMessageBodyPart, msg *imap.Message,
	section *imap.BodySectionName) (io.Reader, error) {
	var r io.Reader
	var err error

	r = msg.GetBody(section)

	if r == nil {
		return nil, nil
	}
	r = encodingReader(task.Encoding, r)
	if task.Charset != "" {
		r, err = message.CharsetReader(task.Charset, r)
	}
	if err != nil {
		return nil, err
	}

	return r, err
}

func encodingReader(encoding string, r io.Reader) io.Reader {
	reader := r
	// email parts are encoded as 7bit (plaintext), quoted-printable, or base64
	if strings.EqualFold(encoding, "base64") {
		reader = base64.NewDecoder(base64.StdEncoding, r)
	} else if strings.EqualFold(encoding, "quoted-printable") {
		reader = quotedprintable.NewReader(r)
	}
	return reader
}
