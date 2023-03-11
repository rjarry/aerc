package imap

import (
	"bufio"
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"

	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders,
) {
	toFetch := msg.Uids
	if imapw.config.cacheEnabled && imapw.cache != nil {
		toFetch = imapw.getCachedHeaders(msg)
	}
	if len(toFetch) == 0 {
		imapw.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)},
			nil)
		return
	}
	log.Tracef("Fetching message headers: %v", toFetch)
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
	imapw.handleFetchMessages(msg, toFetch, items,
		func(_msg *imap.Message) error {
			if len(_msg.Body) == 0 {
				// ignore duplicate messages with only flag updates
				return nil
			}
			reader := _msg.GetBody(section)
			if reader == nil {
				return fmt.Errorf("failed to find part: %v", section)
			}
			textprotoHeader, err := textproto.ReadHeader(bufio.NewReader(reader))
			if err != nil {
				return fmt.Errorf("failed to read part header: %w", err)
			}
			header := &mail.Header{Header: message.Header{Header: textprotoHeader}}
			info := &models.MessageInfo{
				BodyStructure: translateBodyStructure(_msg.BodyStructure),
				Envelope:      translateEnvelope(_msg.Envelope),
				Flags:         translateImapFlags(_msg.Flags),
				InternalDate:  _msg.InternalDate,
				RFC822Headers: header,
				Refs:          parse.MsgIDList(header, "references"),
				Uid:           _msg.Uid,
			}
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    info,
			}, nil)
			if imapw.config.cacheEnabled && imapw.cache != nil {
				imapw.cacheHeader(info)
			}
			return nil
		})
}

func (imapw *IMAPWorker) handleFetchMessageBodyPart(
	msg *types.FetchMessageBodyPart,
) {
	log.Tracef("Fetching message %d part: %v", msg.Uid, msg.Part)

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
	partBodySection.Peek = true

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
			if len(_msg.Body) == 0 {
				// ignore duplicate messages with only flag updates
				return nil
			}
			body := _msg.GetBody(&partHeaderSection)
			if body == nil {
				return fmt.Errorf("failed to find part: %v", partHeaderSection)
			}
			h, err := textproto.ReadHeader(bufio.NewReader(body))
			if err != nil {
				return fmt.Errorf("failed to read part header: %w", err)
			}

			part, err := message.New(message.Header{Header: h},
				_msg.GetBody(&partBodySection))
			if message.IsUnknownCharset(err) {
				log.Warnf("unknown charset encountered "+
					"for uid %d", _msg.Uid)
			} else if err != nil {
				return fmt.Errorf("failed to create message reader: %w", err)
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
	msg *types.FetchFullMessages,
) {
	log.Tracef("Fetching full messages: %v", msg.Uids)
	section := &imap.BodySectionName{
		Peek: true,
	}
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchUid,
		section.FetchItem(),
	}
	imapw.handleFetchMessages(msg, msg.Uids, items,
		func(_msg *imap.Message) error {
			if len(_msg.Body) == 0 {
				// ignore duplicate messages with only flag updates
				return nil
			}
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

func (imapw *IMAPWorker) handleFetchMessageFlags(msg *types.FetchMessageFlags) {
	items := []imap.FetchItem{
		imap.FetchFlags,
		imap.FetchUid,
	}
	imapw.handleFetchMessages(msg, msg.Uids, items,
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

func (imapw *IMAPWorker) handleFetchMessages(
	msg types.WorkerMessage, uids []uint32, items []imap.FetchItem,
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
	if err := imapw.client.UidFetch(set, items, messages); err != nil {
		emitErr(err)
		return
	}
	if err := <-done; err != nil {
		emitErr(err)
		return
	}
	imapw.worker.PostMessage(
		&types.Done{Message: types.RespondTo(msg)}, nil)
}
