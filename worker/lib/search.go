package lib

import (
	"io"
	"net/textproto"
	"strings"
	"time"
	"unicode"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
)

type searchCriteria struct {
	Header textproto.MIMEHeader
	Body   []string
	Text   []string

	WithFlags    models.Flags
	WithoutFlags models.Flags

	startDate, endDate time.Time
}

func GetSearchCriteria(args []string) (*searchCriteria, error) {
	criteria := &searchCriteria{Header: make(textproto.MIMEHeader)}

	opts, optind, err := getopt.Getopts(args, "rux:X:bat:H:f:c:d:")
	if err != nil {
		return nil, err
	}
	body := false
	text := false
	for _, opt := range opts {
		switch opt.Option {
		case 'r':
			criteria.WithFlags |= models.SeenFlag
		case 'u':
			criteria.WithoutFlags |= models.SeenFlag
		case 'x':
			criteria.WithFlags |= getParsedFlag(opt.Value)
		case 'X':
			criteria.WithoutFlags |= getParsedFlag(opt.Value)
		case 'H':
			if strings.Contains(opt.Value, ": ") {
				HeaderValue := strings.SplitN(opt.Value, ": ", 2)
				criteria.Header.Add(HeaderValue[0], HeaderValue[1])
			} else {
				log.Errorf("Header is not given properly, must be given in format `Header: Value`")
				continue
			}
		case 'f':
			criteria.Header.Add("From", opt.Value)
		case 't':
			criteria.Header.Add("To", opt.Value)
		case 'c':
			criteria.Header.Add("Cc", opt.Value)
		case 'b':
			body = true
		case 'd':
			start, end, err := parse.DateRange(opt.Value)
			if err != nil {
				log.Errorf("failed to parse start date: %v", err)
				continue
			}
			if !start.IsZero() {
				criteria.startDate = start
			}
			if !end.IsZero() {
				criteria.endDate = end
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

func getParsedFlag(name string) models.Flags {
	var f models.Flags
	switch strings.ToLower(name) {
	case "seen":
		f = models.SeenFlag
	case "answered":
		f = models.AnsweredFlag
	case "flagged":
		f = models.FlaggedFlag
	}
	return f
}

func Search(messages []rfc822.RawMessage, criteria *searchCriteria) ([]uint32, error) {
	requiredParts := getRequiredParts(criteria)

	matchedUids := []uint32{}
	for _, m := range messages {
		success, err := searchMessage(m, criteria, requiredParts)
		if err != nil {
			return nil, err
		} else if success {
			matchedUids = append(matchedUids, m.UID())
		}
	}

	return matchedUids, nil
}

// searchMessage executes the search criteria for the given RawMessage,
// returns true if search succeeded
func searchMessage(message rfc822.RawMessage, criteria *searchCriteria,
	parts MsgParts,
) (bool, error) {
	// setup parts of the message to use in the search
	// this is so that we try to minimise reading unnecessary parts
	var (
		flags  models.Flags
		header *models.MessageInfo
		body   string
		all    string
		err    error
	)

	if parts&FLAGS > 0 {
		flags, err = message.ModelFlags()
		if err != nil {
			return false, err
		}
	}
	if parts&HEADER > 0 || parts&DATE > 0 {
		header, err = rfc822.MessageInfo(message)
		if err != nil {
			return false, err
		}
	}
	if parts&BODY > 0 {
		// TODO: select body properly; this is just an 'all' clone
		reader, err := message.NewReader()
		if err != nil {
			return false, err
		}
		defer reader.Close()
		bytes, err := io.ReadAll(reader)
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
		defer reader.Close()
		bytes, err := io.ReadAll(reader)
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
	if criteria.WithFlags != 0 {
		if !flags.Has(criteria.WithFlags) {
			return false, nil
		}
	}
	if criteria.WithoutFlags != 0 {
		if flags.Has(criteria.WithoutFlags) {
			return false, nil
		}
	}
	if parts&DATE > 0 {
		if date, err := header.RFC822Headers.Date(); err != nil {
			log.Errorf("Failed to get date from header: %v", err)
		} else {
			if !criteria.startDate.IsZero() {
				if date.Before(criteria.startDate) {
					return false, nil
				}
			}
			if !criteria.endDate.IsZero() {
				if date.After(criteria.endDate) {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

// containsSmartCase is a smarter version of strings.Contains for searching.
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
	DATE
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
	if !criteria.startDate.IsZero() || !criteria.endDate.IsZero() {
		required |= DATE
	}
	if criteria.Body != nil && len(criteria.Body) > 0 {
		required |= BODY
	}
	if criteria.Text != nil && len(criteria.Text) > 0 {
		required |= ALL
	}
	if criteria.WithFlags != 0 {
		required |= FLAGS
	}
	if criteria.WithoutFlags != 0 {
		required |= FLAGS
	}

	return required
}
