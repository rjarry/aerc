package lib

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/emersion/go-imap"
)

var (
	atom *regexp.Regexp = regexp.MustCompile("^[a-z0-9!#$%7'*+-/=?^_`{}|~ ]+$")
)

func FormatAddresses(addrs []*imap.Address) string {
	val := bytes.Buffer{}
	for i, addr := range addrs {
		val.WriteString(FormatAddress(addr))
		if i != len(addrs)-1 {
			val.WriteString(", ")
		}
	}
	return val.String()
}

func FormatAddress(addr *imap.Address) string {
	if addr.PersonalName != "" {
		if atom.MatchString(addr.PersonalName) {
			return fmt.Sprintf("%s <%s@%s>",
				addr.PersonalName, addr.MailboxName, addr.HostName)
		} else {
			return fmt.Sprintf("\"%s\" <%s@%s>",
				strings.ReplaceAll(addr.PersonalName, "\"", "'"),
				addr.MailboxName, addr.HostName)
		}
	} else {
		return fmt.Sprintf("<%s@%s>", addr.MailboxName, addr.HostName)
	}
}
