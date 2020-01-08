package models

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
)

// Flag is an abstraction around the different flags which can be present in
// different email backends and represents a flag that we use in the UI.
type Flag int

const (
	// SeenFlag marks a message as having been seen previously
	SeenFlag Flag = iota

	// RecentFlag marks a message as being recent
	RecentFlag

	// AnsweredFlag marks a message as having been replied to
	AnsweredFlag

	// DeletedFlag marks a message as having been deleted
	DeletedFlag

	// FlaggedFlag marks a message with a user flag
	FlaggedFlag
)

type Directory struct {
	Name       string
	Attributes []string
}

type DirectoryInfo struct {
	Name     string
	Flags    []string
	ReadOnly bool

	// The total number of messages in this mailbox.
	Exists int

	// The number of messages not seen since the last time the mailbox was opened.
	Recent int

	// The number of unread messages
	Unseen int
}

// A MessageInfo holds information about the structure of a message
type MessageInfo struct {
	BodyStructure *BodyStructure
	Envelope      *Envelope
	Flags         []Flag
	Labels        []string
	InternalDate  time.Time
	RFC822Headers *mail.Header
	Size          uint32
	Uid           uint32
}

// A MessageBodyPart can be displayed in the message viewer
type MessageBodyPart struct {
	Reader io.Reader
	Uid    uint32
}

// A FullMessage is the entire message
type FullMessage struct {
	Reader io.Reader
	Uid    uint32
}

type BodyStructure struct {
	MIMEType          string
	MIMESubType       string
	Params            map[string]string
	Description       string
	Encoding          string
	Parts             []*BodyStructure
	Disposition       string
	DispositionParams map[string]string
}

//PartAtIndex returns the BodyStructure at the requested index
func (bs *BodyStructure) PartAtIndex(index []int) (*BodyStructure, error) {
	if len(index) == 0 {
		return bs, nil
	}
	cur := index[0]
	rest := index[1:]
	// passed indexes are 1 based, we need to convert back to actual indexes
	curidx := cur - 1
	if curidx < 0 {
		return nil, fmt.Errorf("invalid index, expected 1 based input")
	}

	// no children, base case
	if len(bs.Parts) == 0 {
		if len(rest) != 0 {
			return nil, fmt.Errorf("more index levels given than available")
		}
		if cur == 1 {
			return bs, nil
		} else {
			return nil, fmt.Errorf("invalid index %v for non multipart", cur)
		}
	}

	if cur > len(bs.Parts) {
		return nil, fmt.Errorf("invalid index %v, only have %v children",
			cur, len(bs.Parts))
	}

	return bs.Parts[curidx].PartAtIndex(rest)
}

type Envelope struct {
	Date      time.Time
	Subject   string
	From      []*Address
	ReplyTo   []*Address
	To        []*Address
	Cc        []*Address
	Bcc       []*Address
	MessageId string
}

type Address struct {
	Name    string
	Mailbox string
	Host    string
}

var atom *regexp.Regexp = regexp.MustCompile("^[a-z0-9!#$%7'*+-/=?^_`{}|~ ]+$")

func (a Address) Format() string {
	if a.Name != "" {
		if atom.MatchString(a.Name) {
			return fmt.Sprintf("%s <%s@%s>", a.Name, a.Mailbox, a.Host)
		} else {
			return fmt.Sprintf("\"%s\" <%s@%s>",
				strings.ReplaceAll(a.Name, "\"", "'"),
				a.Mailbox, a.Host)
		}
	} else {
		return fmt.Sprintf("<%s@%s>", a.Mailbox, a.Host)
	}
}

// FormatAddresses formats a list of addresses, separating each by a comma
func FormatAddresses(addrs []*Address) string {
	val := bytes.Buffer{}
	for i, addr := range addrs {
		val.WriteString(addr.Format())
		if i != len(addrs)-1 {
			val.WriteString(", ")
		}
	}
	return val.String()
}

// OriginalMail is helper struct used for reply/forward
type OriginalMail struct {
	Date     string
	From     string
	Text     string
	MIMEType string
}
