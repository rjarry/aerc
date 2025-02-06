package jmap

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"github.com/emersion/go-message/charset"
)

var bodyProperties = []string{
	"blobId",
	"charset",
	"cid",
	"disposition",
	"language",
	"location",
	"name",
	"partId",
	"size",
	"subParts",
	"type",
}

var emailProperties = []string{
	"id",
	"blobId",
	"threadId",
	"mailboxIds",
	"keywords",
	"size",
	"receivedAt",
	"headers",
	"messageId",
	"inReplyTo",
	"references",
	"from",
	"to",
	"cc",
	"bcc",
	"replyTo",
	"subject",
	"bodyStructure",
}

func (w *JMAPWorker) handleFetchMessageHeaders(msg *types.FetchMessageHeaders) error {
	emailIdsToFetch := make([]jmap.ID, 0, len(msg.Uids))
	currentEmails := make([]*email.Email, 0, len(msg.Uids))
	for _, uid := range msg.Uids {
		jid := jmap.ID(uid)
		m, err := w.cache.GetEmail(jid)
		if err != nil {
			// Message wasn't in cache; fetch it
			emailIdsToFetch = append(emailIdsToFetch, jid)
			continue
		}
		currentEmails = append(currentEmails, m)
		// Get the UI updated immediately
		w.w.PostMessage(&types.MessageInfo{
			Message: types.RespondTo(msg),
			Info:    w.translateMsgInfo(m),
		}, nil)
	}

	if len(emailIdsToFetch) > 0 {
		var req jmap.Request

		req.Invoke(&email.Get{
			Account:        w.AccountId(),
			IDs:            emailIdsToFetch,
			Properties:     emailProperties,
			BodyProperties: bodyProperties,
		})

		resp, err := w.Do(&req)
		if err != nil {
			return err
		}

		for _, inv := range resp.Responses {
			switch r := inv.Args.(type) {
			case *email.GetResponse:
				if err = w.cache.PutEmailState(r.State); err != nil {
					w.w.Warnf("PutEmailState: %s", err)
				}
				currentEmails = append(currentEmails, r.List...)
			case *jmap.MethodError:
				return wrapMethodError(r)
			}
		}
	}

	var threadsToFetch []jmap.ID
	for _, eml := range currentEmails {
		thread, err := w.cache.GetThread(eml.ThreadID)
		if err != nil {
			threadsToFetch = append(threadsToFetch, eml.ThreadID)
			continue
		}
		for _, id := range thread {
			m, err := w.cache.GetEmail(id)
			if err != nil {
				// This should never happen. If we have the
				// thread in cache, we will have fetched it
				// already or updated it from the update loop
				w.w.Warnf("Email ID %s from Thread %s not in cache", id, eml.ThreadID)
				continue
			}
			currentEmails = append(currentEmails, m)
			// Get the UI updated immediately
			w.w.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    w.translateMsgInfo(m),
			}, nil)
		}
	}

	threadEmails, err := w.fetchEntireThreads(threadsToFetch)
	if err != nil {
		return err
	}

	for _, m := range threadEmails {
		w.w.PostMessage(&types.MessageInfo{
			Message: types.RespondTo(msg),
			Info:    w.translateMsgInfo(m),
		}, nil)
		if err := w.cache.PutEmail(m.ID, m); err != nil {
			w.w.Warnf("PutEmail: %s", err)
		}
	}

	return nil
}

func (w *JMAPWorker) handleFetchMessageBodyPart(msg *types.FetchMessageBodyPart) error {
	mail, err := w.cache.GetEmail(jmap.ID(msg.Uid))
	if err != nil {
		return fmt.Errorf("bug: unknown message id %s: %w", msg.Uid, err)
	}

	part := mail.BodyStructure
	for i, index := range msg.Part {
		index -= 1 // convert to zero based offset
		if index < len(part.SubParts) {
			part = part.SubParts[index]
		} else {
			return fmt.Errorf(
				"bug: invalid part index[%d]: %v", i, msg.Part)
		}
	}

	buf, err := w.cache.GetBlob(part.BlobID)
	if err != nil {
		rd, err := w.Download(part.BlobID)
		if err != nil {
			return w.wrapDownloadError("part", part.BlobID, err)
		}
		buf, err = io.ReadAll(rd)
		rd.Close()
		if err != nil {
			return err
		}
		if err = w.cache.PutBlob(part.BlobID, buf); err != nil {
			w.w.Warnf("PutBlob: %s", err)
		}
	}
	var reader io.Reader = bytes.NewReader(buf)
	if strings.HasPrefix(part.Type, "text/") && part.Charset != "" {
		r, err := charset.Reader(part.Charset, reader)
		if err != nil {
			w.w.Warnf("charset.Reader: %v", err)
		} else {
			reader = r
		}
	}
	w.w.PostMessage(&types.MessageBodyPart{
		Message: types.RespondTo(msg),
		Part: &models.MessageBodyPart{
			Reader: reader,
			Uid:    msg.Uid,
		},
	}, nil)

	return nil
}

func (w *JMAPWorker) handleFetchFullMessages(msg *types.FetchFullMessages) error {
	for _, uid := range msg.Uids {
		mail, err := w.cache.GetEmail(jmap.ID(uid))
		if err != nil {
			return fmt.Errorf("bug: unknown message id %s: %w", uid, err)
		}
		buf, err := w.cache.GetBlob(mail.BlobID)
		if err != nil {
			rd, err := w.Download(mail.BlobID)
			if err != nil {
				return w.wrapDownloadError("full", mail.BlobID, err)
			}
			buf, err = io.ReadAll(rd)
			rd.Close()
			if err != nil {
				return err
			}
			if err = w.cache.PutBlob(mail.BlobID, buf); err != nil {
				w.w.Warnf("PutBlob: %s", err)
			}
		}
		w.w.PostMessage(&types.FullMessage{
			Message: types.RespondTo(msg),
			Content: &models.FullMessage{
				Reader: bytes.NewReader(buf),
				Uid:    uid,
			},
		}, nil)
	}

	return nil
}

func (w *JMAPWorker) wrapDownloadError(prefix string, blobId jmap.ID, err error) error {
	urlRepl := strings.NewReplacer(
		"{accountId}", string(w.AccountId()),
		"{blobId}", string(blobId),
		"{type}", "application/octet-stream",
		"{name}", "filename",
	)
	url := urlRepl.Replace(w.client.Session.DownloadURL)
	return fmt.Errorf("%s: %q %w", prefix, url, err)
}
