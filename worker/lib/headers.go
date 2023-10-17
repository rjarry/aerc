package lib

import (
	"strings"

	"github.com/emersion/go-message/mail"
)

// LimitHeaders returns a new Header with the specified headers included or
// excluded
func LimitHeaders(hdr *mail.Header, fields []string, exclude bool) *mail.Header {
	fieldMap := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		fieldMap[strings.ToLower(f)] = struct{}{}
	}
	nh := &mail.Header{}
	curFields := hdr.Fields()
	for curFields.Next() {
		key := strings.ToLower(curFields.Key())
		_, present := fieldMap[key]
		// XOR exclude and present. When they are equal, it means we
		// should not add the header to the new header struct
		if exclude == present {
			continue
		}
		nh.Add(key, curFields.Value())
	}
	return nh
}
