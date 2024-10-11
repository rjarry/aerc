//go:build notmuch
// +build notmuch

package notmuch

import (
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt/v2"
)

type queryBuilder struct {
	s string
}

func (q *queryBuilder) and(s string) {
	if len(s) == 0 {
		return
	}
	if len(q.s) != 0 {
		q.s += " and "
	}
	q.s += "(" + s + ")"
}

func (q *queryBuilder) or(s string) {
	if len(s) == 0 {
		return
	}
	if len(q.s) != 0 {
		q.s += " or "
	}
	q.s += "(" + s + ")"
}

func translate(crit *types.SearchCriteria) string {
	if crit == nil {
		return ""
	}
	var base queryBuilder

	// recipients
	var from queryBuilder
	for _, f := range crit.From {
		from.or("from:" + opt.QuoteArg(f))
	}
	if from.s != "" {
		base.and(from.s)
	}

	var to queryBuilder
	for _, t := range crit.To {
		to.or("to:" + opt.QuoteArg(t))
	}
	if to.s != "" {
		base.and(to.s)
	}

	var cc queryBuilder
	for _, c := range crit.Cc {
		cc.or("cc:" + opt.QuoteArg(c))
	}
	if cc.s != "" {
		base.and(cc.s)
	}

	// flags
	for f := range flagToTag {
		if crit.WithFlags.Has(f) {
			base.and(getParsedFlag(f, false))
		}
		if crit.WithoutFlags.Has(f) {
			base.and(getParsedFlag(f, true))
		}
	}

	// dates
	switch {
	case !crit.StartDate.IsZero() && !crit.EndDate.IsZero():
		base.and(fmt.Sprintf("date:@%d..@%d",
			crit.StartDate.Unix(), crit.EndDate.Unix()))
	case !crit.StartDate.IsZero():
		base.and(fmt.Sprintf("date:@%d..", crit.StartDate.Unix()))
	case !crit.EndDate.IsZero():
		base.and(fmt.Sprintf("date:..@%d", crit.EndDate.Unix()))
	}

	// other terms
	if len(crit.Terms) > 0 {
		if crit.SearchBody {
			base.and("body:" + opt.QuoteArg(strings.Join(crit.Terms, " ")))
		} else {
			for _, term := range crit.Terms {
				base.and(term)
			}
		}
	}

	return base.s
}

func getParsedFlag(flag models.Flags, inverse bool) string {
	name := "tag:" + flagToTag[flag]
	if flagToInvert[flag] {
		name = "not " + name
	}
	return name
}
