package jmap

import (
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/thread"
)

func (w *JMAPWorker) fetchEntireThreads(emailIds []*email.Email) ([]*email.Email, error) {
	var req jmap.Request

	if len(emailIds) == 0 {
		return emailIds, nil
	}

	threadsToFetch := make([]jmap.ID, 0, len(emailIds))
	for _, m := range emailIds {
		threadsToFetch = append(threadsToFetch, m.ThreadID)
	}

	threadGetId := req.Invoke(&thread.Get{
		Account: w.AccountId(),
		IDs:     threadsToFetch,
	})

	// Opportunistically fetch all emails in this thread. We could wait for
	// the result, check which ones we don't have, then fetch only those.
	// However we can do this all in a single request which ends up being
	// faster than two requests for most contexts
	req.Invoke(&email.Get{
		Account: w.AccountId(),
		ReferenceIDs: &jmap.ResultReference{
			ResultOf: threadGetId,
			Name:     "Thread/get",
			Path:     "/list/*/emailIds",
		},
		Properties: headersProperties,
	})

	resp, err := w.Do(&req)
	if err != nil {
		return nil, err
	}

	emailsToReturn := make([]*email.Email, 0)
	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *email.GetResponse:
			emailsToReturn = append(emailsToReturn, r.List...)
			if err = w.cache.PutEmailState(r.State); err != nil {
				w.w.Warnf("PutEmailState: %s", err)
			}
		case *jmap.MethodError:
			return nil, wrapMethodError(r)
		}
	}

	return emailsToReturn, nil
}
