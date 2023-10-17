package jmap

import (
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

func (w *JMAPWorker) translateSearch(
	mbox jmap.ID, criteria *types.SearchCriteria,
) email.Filter {
	cond := new(email.FilterCondition)

	if mbox == "" {
		// all mail virtual folder: display all but trash and spam
		var mboxes []jmap.ID
		if id, ok := w.roles[mailbox.RoleJunk]; ok {
			mboxes = append(mboxes, id)
		}
		if id, ok := w.roles[mailbox.RoleTrash]; ok {
			mboxes = append(mboxes, id)
		}
		cond.InMailboxOtherThan = mboxes
	} else {
		cond.InMailbox = mbox
	}
	if criteria == nil {
		return cond
	}

	// dates
	if !criteria.StartDate.IsZero() {
		cond.After = &criteria.StartDate
	}
	if !criteria.EndDate.IsZero() {
		cond.Before = &criteria.EndDate
	}

	// general search terms
	switch {
	case criteria.SearchAll:
		cond.Text = criteria.Terms
	case criteria.SearchBody:
		cond.Body = criteria.Terms
	default:
		cond.Subject = criteria.Terms
	}

	filter := &email.FilterOperator{Operator: jmap.OperatorAND}
	filter.Conditions = append(filter.Conditions, cond)

	// keywords/flags
	for kw := range flagsToKeywords(criteria.WithFlags) {
		filter.Conditions = append(filter.Conditions,
			&email.FilterCondition{HasKeyword: kw})
	}
	for kw := range flagsToKeywords(criteria.WithoutFlags) {
		filter.Conditions = append(filter.Conditions,
			&email.FilterCondition{NotKeyword: kw})
	}

	// recipients
	addrs := &email.FilterOperator{
		Operator: jmap.OperatorOR,
	}
	for _, from := range criteria.From {
		addrs.Conditions = append(addrs.Conditions,
			&email.FilterCondition{From: from})
	}
	for _, to := range criteria.To {
		addrs.Conditions = append(addrs.Conditions,
			&email.FilterCondition{To: to})
	}
	for _, cc := range criteria.Cc {
		addrs.Conditions = append(addrs.Conditions,
			&email.FilterCondition{Cc: cc})
	}
	if len(addrs.Conditions) > 0 {
		filter.Conditions = append(filter.Conditions, addrs)
	}

	// specific headers
	headers := &email.FilterOperator{
		Operator: jmap.OperatorAND,
	}
	for h, values := range criteria.Headers {
		for _, v := range values {
			headers.Conditions = append(headers.Conditions,
				&email.FilterCondition{Header: []string{h, v}})
		}
	}
	if len(headers.Conditions) > 0 {
		filter.Conditions = append(filter.Conditions, headers)
	}

	return filter
}
