package imap

import (
	"errors"
	"strings"

	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/getopt"
)

func parseSearch(args []string) (*imap.SearchCriteria, error) {
	criteria := imap.NewSearchCriteria()

	opts, optind, err := getopt.Getopts(args, "rubax:X:t:H:f:c:")
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
		}
	}
	if text {
		criteria.Text = args[optind:]
	} else if body {
		criteria.Body = args[optind:]
	} else {
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
