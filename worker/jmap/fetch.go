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

var headersProperties = []string{
	"id",
	"blobId",
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
	var req jmap.Request

	ids := make([]jmap.ID, 0, len(msg.Uids))
	for _, uid := range msg.Uids {
		id, ok := w.uidStore.GetKey(uid)
		if !ok {
			return fmt.Errorf("bug: no jmap id for message uid: %v", uid)
		}
		jid := jmap.ID(id)
		m, err := w.cache.GetEmail(jid)
		if err == nil {
			w.w.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    w.translateMsgInfo(m),
			}, nil)
			continue
		}
		ids = append(ids, jid)
	}

	if len(ids) == 0 {
		return nil
	}

	req.Invoke(&email.Get{
		Account:    w.accountId,
		IDs:        ids,
		Properties: headersProperties,
	})

	resp, err := w.Do(&req)
	if err != nil {
		return err
	}

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *email.GetResponse:
			for _, m := range r.List {
				w.w.PostMessage(&types.MessageInfo{
					Message: types.RespondTo(msg),
					Info:    w.translateMsgInfo(m),
				}, nil)
				if err := w.cache.PutEmail(m.ID, m); err != nil {
					w.w.Warnf("PutEmail: %s", err)
				}
			}
			if err = w.cache.PutEmailState(r.State); err != nil {
				w.w.Warnf("PutEmailState: %s", err)
			}
		case *jmap.MethodError:
			return wrapMethodError(r)
		}
	}

	return nil
}

func (w *JMAPWorker) handleFetchMessageBodyPart(msg *types.FetchMessageBodyPart) error {
	id, ok := w.uidStore.GetKey(msg.Uid)
	if !ok {
		return fmt.Errorf("bug: unknown message uid %d", msg.Uid)
	}
	mail, err := w.cache.GetEmail(jmap.ID(id))
	if err != nil {
		return fmt.Errorf("bug: unknown message id %s: %w", id, err)
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
			return fmt.Errorf("charset.Reader: %w", err)
		}
		reader = r
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
		id, ok := w.uidStore.GetKey(uid)
		if !ok {
			return fmt.Errorf("bug: unknown message uid %d", uid)
		}
		mail, err := w.cache.GetEmail(jmap.ID(id))
		if err != nil {
			return fmt.Errorf("bug: unknown message id %s: %w", id, err)
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
		"{accountId}", string(w.accountId),
		"{blobId}", string(blobId),
		"{type}", "application/octet-stream",
		"{name}", "filename",
	)
	url := urlRepl.Replace(w.client.Session.DownloadURL)
	return fmt.Errorf("%s: %q %w", prefix, url, err)
}
