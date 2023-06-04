package jmap

import (
	"fmt"
	"io"
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/emailsubmission"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
	"github.com/emersion/go-message/mail"
)

func (w *JMAPWorker) handleStartSend(msg *types.StartSendingMessage) error {
	reader, writer := io.Pipe()
	send := &jmapSendWriter{writer: writer, done: make(chan error)}

	w.w.PostMessage(&types.MessageWriter{
		Message: types.RespondTo(msg),
		Writer:  send,
	}, nil)

	go func() {
		defer log.PanicHandler()
		defer close(send.done)

		identity, err := w.getSenderIdentity(msg.Header)
		if err != nil {
			send.done <- err
			return
		}

		blob, err := w.Upload(reader)
		if err != nil {
			send.done <- err
			return
		}

		var req jmap.Request

		// Import the blob into drafts
		req.Invoke(&email.Import{
			Account: w.accountId,
			Emails: map[string]*email.EmailImport{
				"aerc": {
					BlobID: blob.ID,
					MailboxIDs: map[jmap.ID]bool{
						w.roles[mailbox.RoleDrafts]: true,
					},
					Keywords: map[string]bool{
						"$draft": true,
						"$seen":  true,
					},
				},
			},
		})

		// Create the submission
		req.Invoke(&emailsubmission.Set{
			Account: w.accountId,
			Create: map[jmap.ID]*emailsubmission.EmailSubmission{
				"sub": {
					IdentityID: identity,
					EmailID:    "#aerc",
				},
			},
			OnSuccessUpdateEmail: map[jmap.ID]jmap.Patch{
				"#sub": {
					"keywords/$draft":               nil,
					w.rolePatch(mailbox.RoleSent):   true,
					w.rolePatch(mailbox.RoleDrafts): nil,
				},
			},
		})

		resp, err := w.Do(&req)
		if err != nil {
			send.done <- err
			return
		}

		for _, inv := range resp.Responses {
			switch r := inv.Args.(type) {
			case *email.ImportResponse:
				if err, ok := r.NotCreated["aerc"]; ok {
					send.done <- wrapSetError(err)
					return
				}
			case *emailsubmission.SetResponse:
				if err, ok := r.NotCreated["sub"]; ok {
					send.done <- wrapSetError(err)
					return
				}
			case *jmap.MethodError:
				send.done <- wrapMethodError(r)
				return
			}
		}
	}()

	return nil
}

type jmapSendWriter struct {
	writer *io.PipeWriter
	done   chan error
}

func (w *jmapSendWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

func (w *jmapSendWriter) Close() error {
	writeErr := w.writer.Close()
	sendErr := <-w.done
	if writeErr != nil {
		return writeErr
	}
	return sendErr
}

func (w *JMAPWorker) getSenderIdentity(header *mail.Header) (jmap.ID, error) {
	from, err := header.AddressList("from")
	if err != nil {
		return "", fmt.Errorf("msg.Header.AddressList: %w", err)
	}
	if len(from) != 1 {
		return "", fmt.Errorf("no from header in message")
	}
	name, domain, _ := strings.Cut(from[0].Address, "@")
	for _, ident := range w.identities {
		n, d, _ := strings.Cut(ident.Email, "@")
		switch {
		case n == name && d == domain:
			fallthrough
		case n == "*" && d == domain:
			return ident.ID, nil
		}
	}
	return "", fmt.Errorf("no identity found for address: %s@%s", name, domain)
}
