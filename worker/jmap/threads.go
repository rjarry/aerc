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

	req.Invoke(&thread.Get{
		Account: w.AccountId(),
		IDs:     threadsToFetch,
	})

	resp, err := w.Do(&req)
	if err != nil {
		return nil, err
	}

	emailsToFetch := make([]jmap.ID, 0)
	emailsToReturn := make([]*email.Email, 0)
	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *thread.GetResponse:
			for _, t := range r.List {
				for _, emailId := range t.EmailIDs {
					m, err := w.cache.GetEmail(emailId)
					if err == nil || m == nil {
						emailsToFetch = append(emailsToFetch, emailId)
					} else {
						emailsToReturn = append(emailsToReturn, m)
					}
				}
			}
			if err = w.cache.PutThreadState(r.State); err != nil {
				w.w.Warnf("PutThreadState: %s", err)
			}
		case *jmap.MethodError:
			return nil, wrapMethodError(r)
		}
	}

	if len(emailsToFetch) == 0 {
		return emailsToReturn, nil
	}

	req.Invoke(&email.Get{
		Account:    w.AccountId(),
		IDs:        emailsToFetch,
		Properties: headersProperties,
	})

	resp, err = w.Do(&req)
	if err != nil {
		return nil, err
	}

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
