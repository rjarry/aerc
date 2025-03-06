package imap

import (
	"strings"

	"github.com/emersion/go-imap"

	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt/v2"
)

func translateSearch(c *types.SearchCriteria) *imap.SearchCriteria {
	criteria := imap.NewSearchCriteria()
	if c == nil {
		return criteria
	}
	criteria.WithFlags = translateFlags(c.WithFlags)
	criteria.WithoutFlags = translateFlags(c.WithoutFlags)

	if !c.StartDate.IsZero() {
		criteria.SentSince = c.StartDate
	}
	if !c.EndDate.IsZero() {
		criteria.SentBefore = c.EndDate
	}
	for k, v := range c.Headers {
		criteria.Header[k] = v
	}
	for _, f := range c.From {
		criteria.Header.Add("From", f)
	}
	for _, t := range c.To {
		criteria.Header.Add("To", t)
	}
	for _, c := range c.Cc {
		criteria.Header.Add("Cc", c)
	}
	terms := opt.LexArgs(strings.Join(c.Terms, " "))
	if terms.Count() > 0 {
		switch {
		case c.SearchAll:
			criteria.Text = terms.Args()
		case c.SearchBody:
			criteria.Body = terms.Args()
		default:
			for _, term := range terms.Args() {
				criteria.Header.Add("Subject", term)
			}
		}
	}
	return criteria
}
