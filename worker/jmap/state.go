package jmap

import (
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
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
