package imap

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/getopt"
)

func parseSearch(args []string) (*imap.SearchCriteria, error) {
	criteria := imap.NewSearchCriteria()

	opts, optind, err := getopt.Getopts(args, "rubtH:f:")
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
		case 'H':
			// TODO
		case 'f':
			criteria.Header.Add("From", opt.Value)
		case 'b':
			body = true
		case 't':
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
