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
	"git.sr.ht/~rockorager/go-jmap/mail/thread"
	"github.com/emersion/go-message/charset"
)

var headersProperties = []string{
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

func (w *JMAPWorker) fetchEmailIdsFromThreads(threadIds []jmap.ID) ([]jmap.ID, error) {
	currentThreadState, err := w.getThreadState()
	if err != nil {
		return nil, err
	}

	// If we can't get the cached mailbox state, at worst, we will just
	// query information we might already know
	cachedThreadState, err := w.cache.GetThreadState()
	if err != nil {
		w.w.Warnf("GetThreadState: %s", err)
	}

	consistentThreadState := currentThreadState == cachedThreadState

	mailIds := make([]jmap.ID, 0)
	getMailIds := func(threadIds []jmap.ID) error {
		var req jmap.Request
		var realIds []jmap.ID

		if len(threadIds) > 0 {
			realIds = threadIds
		} else {
			realIds = []jmap.ID{jmap.ID("00")}
		}

		req.Invoke(&thread.Get{
			Account: w.accountId,
			IDs:     realIds,
		})

		resp, err := w.Do(&req)
		if err != nil {
			return err
		}

		for _, inv := range resp.Responses {
			switch r := inv.Args.(type) {
			case *thread.GetResponse:
				for _, t := range r.List {
					mailIds = append(mailIds, t.EmailIDs...)
				}
			case *jmap.MethodError:
				return wrapMethodError(r)
			}
		}

		return nil
	}

	// If we have a consistent state, check the cache
	if consistentThreadState {
		missingThreadIds := make([]jmap.ID, 0, len(threadIds))
		for _, threadId := range threadIds {
			t, err := w.cache.GetThread(threadId)
			if err != nil {
				w.w.Warnf("GetThread: %s", err)
				missingThreadIds = append(missingThreadIds, threadId)
				continue
			}
			mailIds = append(mailIds, t.EmailIDs...)
		}

		if len(missingThreadIds) > 0 {
			if err := getMailIds(missingThreadIds); err != nil {
				return nil, err
			}
		}
	} else {
		if err := getMailIds(threadIds); err != nil {
			return nil, err
		}
	}

	if err := w.cache.PutThreadState(currentThreadState); err != nil {
		w.w.Warnf("GetThreadState: %s", err)
	}

	return mailIds, nil
}

func (w *JMAPWorker) handleFetchMessageHeaders(msg *types.FetchMessageHeaders) error {
	mailIds := make([]jmap.ID, 0)
	threadIds := make([]jmap.ID, 0, len(msg.Uids))
	for _, uid := range msg.Uids {
		id, ok := w.uidStore.GetKey(uid)
		if !ok {
			return fmt.Errorf("bug: no jmap id for message uid: %v", uid)
		}
		jid := jmap.ID(id)
		m, err := w.cache.GetEmail(jid)
		// TODO: use ID.Valid() when my patch is merged
		if err == nil && len(m.ThreadID) > 0 && len(m.ThreadID) < 256 {
			threadIds = append(threadIds, m.ThreadID)
			w.w.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    w.translateMsgInfo(m),
			}, nil)
			continue
		}
		mailIds = append(mailIds, jid)
	}

	postMessages := func(mailIds []jmap.ID, collectThreadIds bool) error {
		missing := make([]jmap.ID, 0, len(mailIds))
		for _, id := range mailIds {
			m, err := w.cache.GetEmail(id)
			// TODO: use ID.Valid() when my patch is merged
			if err == nil && len(m.ThreadID) > 0 && len(m.ThreadID) < 256 {
				threadIds = append(threadIds, m.ThreadID)
				w.w.PostMessage(&types.MessageInfo{
					Message: types.RespondTo(msg),
					Info:    w.translateMsgInfo(m),
				}, nil)
				continue
			}
			missing = append(missing, id)
		}

		var req jmap.Request
		req.Invoke(&email.Get{
			Account:    w.accountId,
			IDs:        missing,
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

					if collectThreadIds {
						threadIds = append(threadIds, m.ThreadID)
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

	if len(mailIds) > 0 {
		if err := postMessages(mailIds, true); err != nil {
			return err
		}
	}

	additionalMailIds, err := w.fetchEmailIdsFromThreads(threadIds)
	if err != nil {
		return err
	}

	return postMessages(additionalMailIds, false)
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
