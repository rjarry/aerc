package lib

import (
	"io"
	"strings"
	"unicode"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt"
)

func Search(messages []rfc822.RawMessage, criteria *types.SearchCriteria) ([]uint32, error) {
	criteria.PrepareHeader()
	requiredParts := GetRequiredParts(criteria)

	matchedUids := []uint32{}
	for _, m := range messages {
		success, err := SearchMessage(m, criteria, requiredParts)
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
func SearchMessage(message rfc822.RawMessage, criteria *types.SearchCriteria,
	parts MsgParts,
) (bool, error) {
	if criteria == nil {
		return true, nil
	}
	// setup parts of the message to use in the search
	// this is so that we try to minimise reading unnecessary parts
	var (
		flags models.Flags
		info  *models.MessageInfo
		text  string
		err   error
	)

	if parts&FLAGS > 0 {
		flags, err = message.ModelFlags()
		if err != nil {
			return false, err
		}
	}
	if parts&HEADER > 0 || parts&DATE > 0 || (parts&(BODY|ALL)) == 0 {
		info, err = rfc822.MessageInfo(message)
		if err != nil {
			return false, err
		}
	}
	switch {
	case parts&BODY > 0:
		path := lib.FindFirstNonMultipart(info.BodyStructure, nil)
		reader, err := message.NewReader()
		if err != nil {
			return false, err
		}
		defer reader.Close()
		msg, err := rfc822.ReadMessage(reader)
		if err != nil {
			return false, err
		}
		part, err := rfc822.FetchEntityPartReader(msg, path)
		if err != nil {
			return false, err
		}
		bytes, err := io.ReadAll(part)
		if err != nil {
			return false, err
		}
		text = string(bytes)
	case parts&ALL > 0:
		reader, err := message.NewReader()
		if err != nil {
			return false, err
		}
		defer reader.Close()
		bytes, err := io.ReadAll(reader)
		if err != nil {
			return false, err
		}
		text = string(bytes)
	default:
		text = info.Envelope.Subject
	}

	// now search through the criteria
	// implicit AND at the moment so fail fast
	if criteria.Headers != nil {
		for k, v := range criteria.Headers {
			headerValue := info.RFC822Headers.Get(k)
			for _, text := range v {
				if !containsSmartCase(headerValue, text) {
					return false, nil
				}
			}
		}
	}

	args := opt.LexArgs(criteria.Terms)
	for _, searchTerm := range args.Args() {
		if !containsSmartCase(text, searchTerm) {
			return false, nil
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
		if date, err := info.RFC822Headers.Date(); err != nil {
			log.Errorf("Failed to get date from header: %v", err)
		} else {
			if !criteria.StartDate.IsZero() {
				if date.Before(criteria.StartDate) {
					return false, nil
				}
			}
			if !criteria.EndDate.IsZero() {
				if date.After(criteria.EndDate) {
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
func GetRequiredParts(criteria *types.SearchCriteria) MsgParts {
	required := NONE
	if criteria == nil {
		return required
	}
	if len(criteria.Headers) > 0 {
		required |= HEADER
	}
	if !criteria.StartDate.IsZero() || !criteria.EndDate.IsZero() {
		required |= DATE
	}
	if criteria.SearchBody {
		required |= BODY
	}
	if criteria.SearchAll {
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
