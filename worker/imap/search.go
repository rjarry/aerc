package imap

import (
	"git.sr.ht/~sircmpwn/getopt"
	"github.com/emersion/go-imap"
)

func parseSearch(args []string) (*imap.SearchCriteria, error) {
	criteria := imap.NewSearchCriteria()

	opts, optind, err := getopt.Getopts(args, "ruH:")
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'r':
			criteria.WithFlags = append(criteria.WithFlags, imap.SeenFlag)
		case 'u':
			criteria.WithoutFlags = append(criteria.WithoutFlags, imap.SeenFlag)
		case 'H':
			// TODO
		}
	}
	for _, arg := range args[optind:] {
		criteria.Header.Add("Subject", arg)
	}
	return criteria, nil
}
