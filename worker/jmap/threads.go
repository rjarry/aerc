package jmap

import (
	"context"

	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/thread"
)

func (w *JMAPWorker) fetchEntireThreads(ctx context.Context, threads []jmap.ID) ([]*email.Email, error) {
	var req jmap.Request

	if len(threads) == 0 {
		return []*email.Email{}, nil
	}

	threadGetId := req.Invoke(&thread.Get{
		Account: w.AccountId(),
		IDs:     threads,
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
		Properties:     emailProperties,
		BodyProperties: bodyProperties,
	})

	resp, err := w.Do(ctx, &req)
	if err != nil {
		return nil, err
	}

	emailsToReturn := make([]*email.Email, 0)
	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *thread.GetResponse:
			if err = w.cache.PutThreadState(r.State); err != nil {
				w.w.Warnf("PutThreadState: %s", err)
			}
			for _, thread := range r.List {
				if err = w.cache.PutThread(thread.ID, thread.EmailIDs); err != nil {
					w.w.Warnf("PutThread: %s", err)
				}
			}
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
