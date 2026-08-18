package parse

import (
	"regexp"
	"strings"

	"github.com/emersion/go-message/mail"
)

var msgIdRe = regexp.MustCompile(`<(\w[^@>]*@[^@>]*\w)>`)

func MsgID(h *mail.Header) string {
	txt, _ := h.Text("Message-ID")
	m := msgIdRe.FindStringSubmatch(txt)
	if m == nil {
		return ""
	}
	return m[1]
}

// MsgIDList parses a list of message identifiers.  It returns message
// identifiers without angle brackets.
//
// This can be used on In-Reply-To and References header fields.
func MsgIDList(h *mail.Header, key string) []string {
	txt, _ := h.Text(key)
	matches := msgIdRe.FindAllStringSubmatch(txt, -1)
	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, m[1])
	}
	return ids
}

// Mailto parses a URI string and extracts the mail address.
// Example: "<mailto:user@domain.com>" -> "user@domain.com"
func Mailto(val string) ([]*mail.Address, error) {
	s := strings.Trim(val, "<> \t\r\n")
	return mail.ParseAddressList(strings.TrimPrefix(s, "mailto:"))
}
