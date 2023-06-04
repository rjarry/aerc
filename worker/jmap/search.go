package jmap

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~sircmpwn/getopt"
)

func parseSearch(args []string) (*email.FilterCondition, error) {
	f := new(email.FilterCondition)
	if len(args) == 0 {
		return f, nil
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
			f.HasKeyword = "$seen"
		case 'u':
			f.NotKeyword = "$seen"
		case 'f':
			f.From = opt.Value
		case 't':
			f.To = opt.Value
		case 'c':
			f.Cc = opt.Value
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
				f.After = &start
			}
			if !end.IsZero() {
				f.Before = &end
			}
		}
	}
	switch {
	case text:
		f.Text = strings.Join(args[optind:], " ")
	case body:
		f.Body = strings.Join(args[optind:], " ")
	default:
		f.Subject = strings.Join(args[optind:], " ")
	}
	return f, nil
}
