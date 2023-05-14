//go:build notmuch
// +build notmuch

package notmuch

import (
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~sircmpwn/getopt"
)

type queryBuilder struct {
	s string
}

func (q *queryBuilder) add(s string) {
	if len(s) == 0 {
		return
	}
	if len(q.s) != 0 {
		q.s += " and "
	}
	q.s += "(" + s + ")"
}

func translate(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	var qb queryBuilder
	opts, optind, err := getopt.Getopts(args, "rux:X:bat:H:f:c:d:")
	if err != nil {
		// if error occurs here, don't fail
		log.Errorf("getopts failed: %v", err)
		return strings.Join(args[1:], ""), nil
	}
	body := false
	for _, opt := range opts {
		switch opt.Option {
		case 'r':
			qb.add("not tag:unread")
		case 'u':
			qb.add("tag:unread")
		case 'x':
			qb.add(getParsedFlag(opt.Value))
		case 'X':
			qb.add("not " + getParsedFlag(opt.Value))
		case 'H':
			// TODO
		case 'f':
			qb.add("from:" + opt.Value)
		case 't':
			qb.add("to:" + opt.Value)
		case 'c':
			qb.add("cc:" + opt.Value)
		case 'a':
			// TODO
		case 'b':
			body = true
		case 'd':
			qb.add("date:" + strconv.Quote(opt.Value))
		}
	}
	switch {
	case body:
		qb.add("body:" + strconv.Quote(strings.Join(args[optind:], " ")))
	default:
		qb.add(strings.Join(args[optind:], " "))
	}
	return qb.s, nil
}

func getParsedFlag(name string) string {
	switch strings.ToLower(name) {
	case "answered":
		return "tag:replied"
	case "seen":
		return "(not tag:unread)"
	case "flagged":
		return "tag:flagged"
	default:
		return name
	}
}
