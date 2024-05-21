package jmap

import (
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
	"git.sr.ht/~rockorager/go-jmap/mail/thread"
)

func (w *JMAPWorker) getMailboxState() (string, error) {
	var req jmap.Request

	req.Invoke(&mailbox.Get{Account: w.accountId, IDs: make([]jmap.ID, 0)})
	resp, err := w.Do(&req)
	if err != nil {
		return "", err
	}

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *mailbox.GetResponse:
			return r.State, nil
		case *jmap.MethodError:
			return "", wrapMethodError(r)

		}
	}

	// This should be an impossibility
	return "", nil
}

func (w *JMAPWorker) getThreadState() (string, error) {
	var req jmap.Request

	// TODO: This is a junk JMAP ID because Go's JSON serialization doesn't
	// send empty slices as arrays, WTF.
	req.Invoke(&thread.Get{Account: w.accountId, IDs: []jmap.ID{jmap.ID("00")}})
	resp, err := w.Do(&req)
	if err != nil {
		return "", err
	}

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *thread.GetResponse:
			return r.State, nil
		case *jmap.MethodError:
			return "", wrapMethodError(r)

		}
	}

	// This should be an impossibility
	return "", nil
}
