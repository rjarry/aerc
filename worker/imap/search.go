package imap

import (
	"errors"
	"strings"

	"github.com/emersion/go-imap"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"git.sr.ht/~sircmpwn/getopt"
)

func parseSearch(args []string) (*imap.SearchCriteria, error) {
	criteria := imap.NewSearchCriteria()
	if len(args) == 0 {
		return criteria, nil
	}

	opts, optind, err := getopt.Getopts(args, "rubax:X:t:H:f:c:d:")
	if err != nil {
		return nil, err
	}
	body := false
	text := false
	for _, opt := range opts {
		switch opt.Option {
		case 'r':
			criteria.WithFlags = append(criteria.WithFlags, imap.SeenFlag)
		case 'u':
			criteria.WithoutFlags = append(criteria.WithoutFlags, imap.SeenFlag)
		case 'x':
			if f, err := getParsedFlag(opt.Value); err == nil {
				criteria.WithFlags = append(criteria.WithFlags, f)
			}
		case 'X':
			if f, err := getParsedFlag(opt.Value); err == nil {
				criteria.WithoutFlags = append(criteria.WithoutFlags, f)
			}
		case 'H':
			// TODO
		case 'f':
			criteria.Header.Add("From", opt.Value)
		case 't':
			criteria.Header.Add("To", opt.Value)
		case 'c':
			criteria.Header.Add("Cc", opt.Value)
		case 'b':
			body = true
		case 'a':
			text = true
		case 'd':
			start, end, err := lib.ParseDateRange(opt.Value)
			if err != nil {
				log.Errorf("failed to parse start date: %v", err)
				continue
			}
			if !start.IsZero() {
				criteria.SentSince = start
			}
			if !end.IsZero() {
				criteria.SentBefore = end
			}
		}
	}
	switch {
	case text:
		criteria.Text = args[optind:]
	case body:
		criteria.Body = args[optind:]
	default:
		for _, arg := range args[optind:] {
			criteria.Header.Add("Subject", arg)
		}
	}
	return criteria, nil
}

func getParsedFlag(name string) (string, error) {
	switch strings.ToLower(name) {
	case "seen":
		return imap.SeenFlag, nil
	case "flagged":
		return imap.FlaggedFlag, nil
	case "answered":
		return imap.AnsweredFlag, nil
	}
	return imap.FlaggedFlag, errors.New("Flag not suppored")
}
