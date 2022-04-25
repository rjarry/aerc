package models

import (
	"errors"
	"fmt"
	"io"
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

	// set to true if the value counts are accurate
	AccurateCounts bool
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
	Error         error
}

func (mi *MessageInfo) MsgId() (msgid string, err error) {
	if mi == nil {
		return "", errors.New("msg is nil")
	}
	if mi.Envelope == nil {
		return "", errors.New("envelope is nil")
	}
	return mi.Envelope.MessageId, nil
}

func (mi *MessageInfo) InReplyTo() (msgid string, err error) {
	if mi == nil {
		return "", errors.New("msg is nil")
	}
	if mi.RFC822Headers == nil {
		return "", errors.New("header is nil")
	}
	list, err := mi.RFC822Headers.MsgIDList("In-Reply-To")
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", errors.New("no results")
	}
	return list[0], err
}

func (mi *MessageInfo) References() ([]string, error) {
	if mi == nil {
		return []string{}, errors.New("msg is nil")
	}
	if mi.RFC822Headers == nil {
		return []string{}, errors.New("header is nil")
	}
	list, err := mi.RFC822Headers.MsgIDList("References")
	if err != nil {
		return []string{}, err
	}
	if len(list) == 0 {
		return []string{}, errors.New("no results")
	}
	return list, err
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
	From      []*mail.Address
	ReplyTo   []*mail.Address
	To        []*mail.Address
	Cc        []*mail.Address
	Bcc       []*mail.Address
	MessageId string
}

// OriginalMail is helper struct used for reply/forward
type OriginalMail struct {
	Date          time.Time
	From          string
	Text          string
	MIMEType      string
	RFC822Headers *mail.Header
}

type SignatureValidity int32

const (
	UnknownValidity SignatureValidity = iota
	Valid
	InvalidSignature
	UnknownEntity
	UnsupportedMicalg
	MicalgMismatch
)

type MessageDetails struct {
	IsEncrypted        bool
	IsSigned           bool
	SignedBy           string // Primary identity of signing key
	SignedByKeyId      uint64
	SignatureValidity  SignatureValidity
	SignatureError     string
	DecryptedWith      string // Primary Identity of decryption key
	DecryptedWithKeyId uint64 // Public key id of decryption key
	Body               io.Reader
	Micalg             string
}
