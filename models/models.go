package models

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/parse"
	"github.com/emersion/go-message/mail"
)

// Flags is an abstraction around the different flags which can be present in
// different email backends and represents a flag that we use in the UI.
type Flags uint32

const (
	// SeenFlag marks a message as having been seen previously
	SeenFlag Flags = 1 << iota

	// RecentFlag marks a message as being recent
	RecentFlag

	// AnsweredFlag marks a message as having been replied to
	AnsweredFlag

	// ForwardedFlag marks a message as having been forwarded
	ForwardedFlag

	// DeletedFlag marks a message as having been deleted
	DeletedFlag

	// FlaggedFlag marks a message with a user flag
	FlaggedFlag

	// DraftFlag marks a message as a draft
	DraftFlag
)

func (f Flags) Has(flags Flags) bool {
	return f&flags == flags
}

type Role string

var Roles = map[string]Role{
	"all":     AllRole,
	"archive": ArchiveRole,
	"drafts":  DraftsRole,
	"inbox":   InboxRole,
	"junk":    JunkRole,
	"sent":    SentRole,
	"trash":   TrashRole,
	"query":   QueryRole,
}

const (
	AllRole     Role = "all"
	ArchiveRole Role = "archive"
	DraftsRole  Role = "drafts"
	InboxRole   Role = "inbox"
	JunkRole    Role = "junk"
	SentRole    Role = "sent"
	TrashRole   Role = "trash"
	// Custom aerc roles
	QueryRole Role = "query"
	// virtual node created by the directory tree
	VirtualRole Role = "virtual"
)

type Directory struct {
	Name string
	// Exists messages in the Directory
	Exists int
	// Recent messages in the Directory
	Recent int
	// Unseen messages in the Directory
	Unseen int
	// IANA role
	Role Role
}

type DirectoryInfo struct {
	Name string
	// The total number of messages in this mailbox.
	Exists int
	// The number of messages not seen since the last time the mailbox was opened.
	Recent int
	// The number of unread messages
	Unseen int
}

// Capabilities provides the backend capabilities
type Capabilities struct {
	Sort       bool
	Thread     bool
	Extensions []string
}

func (c *Capabilities) Has(s string) bool {
	return slices.Contains(c.Extensions, s)
}

type UID string

func UidToUint32(uid UID) uint32 {
	u, _ := strconv.ParseUint(string(uid), 10, 32)
	return uint32(u)
}

func Uint32ToUid(u uint32) UID {
	return UID(fmt.Sprintf("%012d", u))
}

func UidToUint32List(uids []UID) []uint32 {
	ulist := make([]uint32, 0, len(uids))
	for _, uid := range uids {
		ulist = append(ulist, UidToUint32(uid))
	}
	return ulist
}

func Uint32ToUidList(ulist []uint32) []UID {
	uids := make([]UID, 0, len(ulist))
	for _, u := range ulist {
		uids = append(uids, Uint32ToUid(u))
	}
	return uids
}

// A MessageInfo holds information about the structure of a message
type MessageInfo struct {
	BodyStructure *BodyStructure
	Envelope      *Envelope
	Flags         Flags
	Labels        []string
	Filenames     []string
	InternalDate  time.Time
	RFC822Headers *mail.Header
	Refs          []string
	Size          uint32
	Uid           UID
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
	if mi.Envelope != nil && mi.Envelope.InReplyTo != "" {
		return mi.Envelope.InReplyTo, nil
	}
	if mi.RFC822Headers == nil {
		return "", errors.New("header is nil")
	}
	list := parse.MsgIDList(mi.RFC822Headers, "In-Reply-To")
	if len(list) == 0 {
		return "", errors.New("no results")
	}
	return list[0], err
}

func (mi *MessageInfo) References() ([]string, error) {
	if mi == nil {
		return []string{}, errors.New("msg is nil")
	}
	if mi.Refs != nil {
		return mi.Refs, nil
	}
	if mi.RFC822Headers == nil {
		return []string{}, errors.New("header is nil")
	}
	list := parse.MsgIDList(mi.RFC822Headers, "References")
	if len(list) == 0 {
		return []string{}, errors.New("no results")
	}
	return list, nil
}

// A MessageBodyPart can be displayed in the message viewer
type MessageBodyPart struct {
	Reader io.Reader
	Uid    UID
}

// A FullMessage is the entire message
type FullMessage struct {
	Reader io.Reader
	Uid    UID
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

// PartAtIndex returns the BodyStructure at the requested index
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

func (bs *BodyStructure) FullMIMEType() string {
	mime := fmt.Sprintf("%s/%s", bs.MIMEType, bs.MIMESubType)
	return strings.ToLower(mime)
}

func (bs *BodyStructure) FileName() string {
	if filename, ok := bs.DispositionParams["filename"]; ok {
		return filename
	} else if filename, ok := bs.Params["name"]; ok {
		// workaround golang not supporting RFC2231 besides ASCII and UTF8
		return filename
	}
	return ""
}

type Envelope struct {
	Date      time.Time
	Subject   string
	From      []*mail.Address
	ReplyTo   []*mail.Address
	Sender    []*mail.Address
	To        []*mail.Address
	Cc        []*mail.Address
	Bcc       []*mail.Address
	MessageId string
	InReplyTo string
}

// OriginalMail is helper struct used for reply/forward
type OriginalMail struct {
	Date          time.Time
	From          string
	Text          string
	MIMEType      string
	RFC822Headers *mail.Header
	Folder        string
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
