package parse

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/emersion/go-message/mail"
)

// MsgIDList parses a list of message identifiers.  It returns message
// identifiers without angle brackets.  If the header field is missing,
// it returns nil.
//
// This can be used on In-Reply-To and References header fields.
// If the field does not conform to RFC 5322, fall back
// to greedily parsing a subsequence of the original field.
func MsgIDList(h *mail.Header, key string) []string {
	l, err := h.MsgIDList(key)
	if err == nil {
		return l
	}
	log.Errorf("%s: %s", err, h.Get(key))

	// Expensive, fix your peer's MUA instead!
	var list []string
	header := &mail.Header{Header: h.Header.Copy()}
	value := header.Get(key)
	for err != nil && len(value) > 0 {
		// Skip parsed IDs
		if len(l) > 0 {
			last := "<" + l[len(l)-1] + ">"
			value = value[strings.Index(value, last)+len(last):]
			list = append(list, l...)
		}

		// Skip a character until some IDs can be parsed
		value = value[1:]
		header.Set(key, value)
		l, err = header.MsgIDList(key)
	}
	return append(list, l...)
}
