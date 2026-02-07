package imap

import (
	"bufio"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-imap/utf7"
)

func (imapw *IMAPWorker) attachGMLabels(_msg *imap.Message, info *models.MessageInfo) {
	if len(_msg.Items["X-GM-LABELS"].([]any)) == 0 {
		return
	}
	imapw.worker.Debugf("Attaching labels %v to message %v\n", _msg.Items["X-GM-LABELS"], _msg.Uid)
	enc := utf7.Encoding.NewDecoder()
	for _, label := range _msg.Items["X-GM-LABELS"].([]any) {
		decodedLabel, err := enc.String(label.(string))
		if err != nil {
			imapw.worker.Errorf("Failed to decode label %v from UTF-7\n", label.(string))
			info.Labels = append(info.Labels, label.(string))
		} else {
			info.Labels = append(info.Labels, decodedLabel)
		}
	}
}

func (imapw *IMAPWorker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders,
) error {
	if msg.Context().Err() != nil {
		return msg.Context().Err()
	}
	toFetch := msg.Uids

	// Expand UIDs to include entire threads when Gmail X-GM-EXT-1 is available
	if imapw.caps.Has("X-GM-EXT-1") {
		uids, err := imapw.fetchEntireThreads(toFetch)
		if err != nil {
			imapw.worker.Warnf("failed to fetch entire threads: %v", err)
		} else if len(uids) > 0 {
			toFetch = uids
		}
	}

	cacheEnabled := imapw.config.cacheEnabled && imapw.cache != nil
	if cacheEnabled {
		msg.Uids = toFetch
		toFetch = imapw.getCachedHeaders(msg)
	}
	if len(toFetch) == 0 {
		return nil
	}
	imapw.worker.Tracef("Fetching message headers: %v", toFetch)
	hdrBodyPart := imap.BodyPartName{
		Specifier: imap.HeaderSpecifier,
	}
	switch {
	case len(imapw.config.headersExclude) > 0:
		hdrBodyPart.NotFields = true
		hdrBodyPart.Fields = imapw.config.headersExclude
	case len(imapw.config.headers) > 0:
		hdrBodyPart.Fields = imapw.config.headers
	}
	section := &imap.BodySectionName{
		BodyPartName: hdrBodyPart,
		Peek:         true,
	}

	items := []imap.FetchItem{
		imap.FetchBodyStructure,
		imap.FetchEnvelope,
		imap.FetchInternalDate,
		imap.FetchFlags,
		imap.FetchUid,
		imap.FetchRFC822Size,
		section.FetchItem(),
	}

	return imapw.handleFetchMessages(msg.Context(), msg, toFetch, items,
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
			systemFlags, keywordFlags := translateImapFlags(_msg.Flags)
			info := &models.MessageInfo{
				BodyStructure: translateBodyStructure(_msg.BodyStructure),
				Envelope:      translateEnvelope(_msg.Envelope),
				Flags:         systemFlags,
				InternalDate:  _msg.InternalDate,
				RFC822Headers: header,
				Refs:          parse.MsgIDList(header, "references"),
				Size:          _msg.Size,
				Uid:           models.Uint32ToUid(_msg.Uid),
			}

			if imapw.caps.Has("X-GM-EXT-1") {
				imapw.attachGMLabels(_msg, info)
			} else if len(imapw.selected.PermanentFlags) == 0 || slices.Contains(imapw.selected.PermanentFlags, "\\*") {
				info.Labels = keywordFlags
			}

			if cacheEnabled && !info.Flags.Has(models.SeenFlag) &&
				time.Since(info.InternalDate) < imapw.config.checkMail {
				// Consider unread messages received within the last CheckMail
				// period as Recent, regardless of what the IMAP server says.
				info.Flags |= models.RecentFlag
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
) error {
	imapw.worker.Tracef("Fetching message %d part: %v", msg.Uid, msg.Part)

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
	return imapw.handleFetchMessages(msg.Context(), msg, []models.UID{msg.Uid}, items,
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
				imapw.worker.Warnf("unknown charset encountered "+
					"for uid %d", _msg.Uid)
			} else if err != nil {
				return fmt.Errorf("failed to create message reader: %w", err)
			}

			imapw.worker.PostMessage(&types.MessageBodyPart{
				Message: types.RespondTo(msg),
				Part: &models.MessageBodyPart{
					Reader: part.Body,
					Uid:    models.Uint32ToUid(_msg.Uid),
				},
			}, nil)
			// Update flags (to mark message as read)
			systemFlags, keywordFlags := translateImapFlags(_msg.Flags)
			info := &models.MessageInfo{
				Flags: systemFlags,
				Uid:   models.Uint32ToUid(_msg.Uid),
			}
			if imapw.caps.Has("X-GM-EXT-1") {
				imapw.attachGMLabels(_msg, info)
			} else if len(imapw.selected.PermanentFlags) == 0 || slices.Contains(imapw.selected.PermanentFlags, "\\*") {
				info.Labels = keywordFlags
			}
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    info,
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleFetchFullMessages(
	msg *types.FetchFullMessages,
) error {
	imapw.worker.Tracef("Fetching full messages: %v", msg.Uids)
	section := &imap.BodySectionName{
		Peek: true,
	}
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchUid,
		section.FetchItem(),
	}
	return imapw.handleFetchMessages(msg.Context(), msg, msg.Uids, items,
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
					Uid:    models.Uint32ToUid(_msg.Uid),
				},
			}, nil)
			// Update flags (to mark message as read)
			systemFlags, keywordFlags := translateImapFlags(_msg.Flags)
			info := &models.MessageInfo{
				Flags: systemFlags,
				Uid:   models.Uint32ToUid(_msg.Uid),
			}
			if imapw.caps.Has("X-GM-EXT-1") {
				imapw.attachGMLabels(_msg, info)
			} else if len(imapw.selected.PermanentFlags) == 0 || slices.Contains(imapw.selected.PermanentFlags, "\\*") {
				info.Labels = keywordFlags
			}
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    info,
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleFetchMessageFlags(msg *types.FetchMessageFlags) error {
	items := []imap.FetchItem{
		imap.FetchFlags,
		imap.FetchUid,
	}
	return imapw.handleFetchMessages(msg.Context(), msg, msg.Uids, items,
		func(_msg *imap.Message) error {
			systemFlags, keywordFlags := translateImapFlags(_msg.Flags)
			info := &models.MessageInfo{
				Flags: systemFlags,
				Uid:   models.Uint32ToUid(_msg.Uid),
			}

			if imapw.caps.Has("X-GM-EXT-1") {
				imapw.attachGMLabels(_msg, info)
			} else if len(imapw.selected.PermanentFlags) == 0 || slices.Contains(imapw.selected.PermanentFlags, "\\*") {
				info.Labels = keywordFlags
			}

			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    info,
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleFetchMessages(
	ctx context.Context, msg types.WorkerMessage, uids []models.UID,
	items []imap.FetchItem, procFunc func(*imap.Message) error,
) error {
	messages := make(chan *imap.Message)
	done := make(chan error)

	missingUids := make(map[models.UID]bool)
	for _, uid := range uids {
		missingUids[uid] = true
	}

	go func() {
		defer log.PanicHandler()

		for {
			select {
			case <-ctx.Done():
				done <- ctx.Err()
				goto out
			case _msg := <-messages:
				if _msg == nil {
					goto out
				}
				delete(missingUids, models.Uint32ToUid(_msg.Uid))
				err := procFunc(_msg)
				if err != nil {
					imapw.worker.Errorf("failed to process message <%d>: %v", _msg.Uid, err)
					imapw.worker.PostMessage(&types.MessageInfo{
						Message: types.RespondTo(msg),
						Info: &models.MessageInfo{
							Uid:   models.Uint32ToUid(_msg.Uid),
							Error: err,
						},
					}, nil)
				}
			}
		}
	out:
		close(done)
	}()

	if imapw.caps.Has("X-GM-EXT-1") {
		items = append(items, "X-GM-LABELS")
	}

	set := toSeqSet(uids)
	if err := imapw.client.UidFetch(set, items, messages); err != nil {
		return err
	}
	err := <-done
	if err != nil {
		return err
	}

	for uid := range missingUids {
		imapw.worker.PostMessage(&types.MessageInfo{
			Message: types.RespondTo(msg),
			Info: &models.MessageInfo{
				Uid:   uid,
				Error: fmt.Errorf("invalid response from server (detailed error in log)"),
			},
		}, nil)
	}

	return nil
}
