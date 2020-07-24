package maildir

import (
	"io/ioutil"
	"net/textproto"
	"strings"
	"unicode"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~sircmpwn/aerc/models"
)

type searchCriteria struct {
	Header textproto.MIMEHeader
	Body   []string
	Text   []string

	WithFlags    []maildir.Flag
	WithoutFlags []maildir.Flag
}

func newSearchCriteria() *searchCriteria {
	return &searchCriteria{Header: make(textproto.MIMEHeader)}
}

func parseSearch(args []string) (*searchCriteria, error) {
	criteria := newSearchCriteria()

	opts, optind, err := getopt.Getopts(args, "rux:X:bat:H:f:c:")
	if err != nil {
		return nil, err
	}
	body := false
	text := false
	for _, opt := range opts {
		switch opt.Option {
		case 'r':
			criteria.WithFlags = append(criteria.WithFlags, maildir.FlagSeen)
		case 'u':
			criteria.WithoutFlags = append(criteria.WithoutFlags, maildir.FlagSeen)
		case 'x':
			criteria.WithFlags = append(criteria.WithFlags, getParsedFlag(opt.Value))
		case 'X':
			criteria.WithoutFlags = append(criteria.WithoutFlags, getParsedFlag(opt.Value))
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

func getParsedFlag(name string) maildir.Flag {
	var f maildir.Flag
	switch strings.ToLower(name) {
	case "seen":
		f = maildir.FlagSeen
	case "answered":
		f = maildir.FlagReplied
	case "flagged":
		f = maildir.FlagFlagged
	}
	return f
}

func (w *Worker) search(criteria *searchCriteria) ([]uint32, error) {
	requiredParts := getRequiredParts(criteria)
	w.worker.Logger.Printf("Required parts bitmask for search: %b", requiredParts)

	keys, err := w.c.UIDs(*w.selected)
	if err != nil {
		return nil, err
	}

	matchedUids := []uint32{}
	for _, key := range keys {
		success, err := w.searchKey(key, criteria, requiredParts)
		if err != nil {
			// don't return early so that we can still get some results
			w.worker.Logger.Printf("Failed to search key %v: %v", key, err)
		} else if success {
			matchedUids = append(matchedUids, key)
		}
	}

	return matchedUids, nil
}

// Execute the search criteria for the given key, returns true if search succeeded
func (w *Worker) searchKey(key uint32, criteria *searchCriteria,
	parts MsgParts) (bool, error) {
	message, err := w.c.Message(*w.selected, key)
	if err != nil {
		return false, err
	}

	// setup parts of the message to use in the search
	// this is so that we try to minimise reading unnecessary parts
	var (
		flags  []maildir.Flag
		header *models.MessageInfo
		body   string
		all    string
	)

	if parts&FLAGS > 0 {
		flags, err = message.Flags()
		if err != nil {
			return false, err
		}
	}
	if parts&HEADER > 0 {
		header, err = message.MessageInfo()
		if err != nil {
			return false, err
		}
	}
	if parts&BODY > 0 {
		// TODO: select which part to search, maybe look for text/plain
		reader, err := message.NewBodyPartReader([]int{1})
		if err != nil {
			return false, err
		}
		bytes, err := ioutil.ReadAll(reader)
		if err != nil {
			return false, err
		}
		body = string(bytes)
	}
	if parts&ALL > 0 {
		reader, err := message.NewReader()
		if err != nil {
			return false, err
		}
		bytes, err := ioutil.ReadAll(reader)
		if err != nil {
			return false, err
		}
		all = string(bytes)
	}

	// now search through the criteria
	// implicit AND at the moment so fail fast
	if criteria.Header != nil {
		for k, v := range criteria.Header {
			headerValue := header.RFC822Headers.Get(k)
			for _, text := range v {
				if !containsSmartCase(headerValue, text) {
					return false, nil
				}
			}
		}
	}
	if criteria.Body != nil {
		for _, searchTerm := range criteria.Body {
			if !containsSmartCase(body, searchTerm) {
				return false, nil
			}
		}
	}
	if criteria.Text != nil {
		for _, searchTerm := range criteria.Text {
			if !containsSmartCase(all, searchTerm) {
				return false, nil
			}
		}
	}
	if criteria.WithFlags != nil {
		for _, searchFlag := range criteria.WithFlags {
			if !containsFlag(flags, searchFlag) {
				return false, nil
			}
		}
	}
	if criteria.WithoutFlags != nil {
		for _, searchFlag := range criteria.WithoutFlags {
			if containsFlag(flags, searchFlag) {
				return false, nil
			}
		}
	}
	return true, nil
}

// Returns true if searchFlag appears in flags
func containsFlag(flags []maildir.Flag, searchFlag maildir.Flag) bool {
	match := false
	for _, flag := range flags {
		if searchFlag == flag {
			match = true
		}
	}
	return match
}

// Smarter version of strings.Contains for searching.
// Is case-insensitive unless substr contains an upper case character
func containsSmartCase(s string, substr string) bool {
	if hasUpper(substr) {
		return strings.Contains(s, substr)
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// The parts of a message, kind of
type MsgParts int

const NONE MsgParts = 0
const (
	FLAGS MsgParts = 1 << iota
	HEADER
	BODY
	ALL
)

// Returns a bitmask of the parts of the message required to be loaded for the
// given criteria
func getRequiredParts(criteria *searchCriteria) MsgParts {
	required := NONE
	if len(criteria.Header) > 0 {
		required |= HEADER
	}
	if criteria.Body != nil && len(criteria.Body) > 0 {
		required |= BODY
	}
	if criteria.Text != nil && len(criteria.Text) > 0 {
		required |= ALL
	}
	if criteria.WithFlags != nil && len(criteria.WithFlags) > 0 {
		required |= FLAGS
	}
	if criteria.WithoutFlags != nil && len(criteria.WithoutFlags) > 0 {
		required |= FLAGS
	}

	return required
}
