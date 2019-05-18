package types

import (
	"crypto/x509"
	"io"
	"time"

	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc/config"
)

type WorkerMessage interface {
	InResponseTo() WorkerMessage
	getId() int
	setId(id int)
}

type Message struct {
	inResponseTo WorkerMessage
	id           int
}

func RespondTo(msg WorkerMessage) Message {
	return Message{
		inResponseTo: msg,
	}
}

func (m Message) InResponseTo() WorkerMessage {
	return m.inResponseTo
}

func (m Message) getId() int {
	return m.id
}

func (m Message) setId(id int) {
	m.id = id
}

// Meta-messages

type Done struct {
	Message
}

type Error struct {
	Message
	Error error
}

type Unsupported struct {
	Message
}

// Actions

type ApproveCertificate struct {
	Message
	Approved bool
}

type Configure struct {
	Message
	Config *config.AccountConfig
}

type Connect struct {
	Message
}

type Disconnect struct {
	Message
}

type ListDirectories struct {
	Message
}

type OpenDirectory struct {
	Message
	Directory string
}

type FetchDirectoryContents struct {
	Message
}

type FetchMessageHeaders struct {
	Message
	Uids imap.SeqSet
}

type FetchFullMessages struct {
	Message
	Uids imap.SeqSet
}

type FetchMessageBodyPart struct {
	Message
	Uid  uint32
	Part int
}

type DeleteMessages struct {
	Message
	Uids imap.SeqSet
}

type CopyMessages struct {
	Message
	Destination string
	Uids        imap.SeqSet
}

type AppendMessage struct {
	Message
	Destination string
	Flags       []string
	Date        time.Time
	Reader      io.Reader
	Length      int
}

// Messages

type CertificateApprovalRequest struct {
	Message
	CertPool *x509.CertPool
}

type Directory struct {
	Message
	Attributes []string
	Name       string
}

type DirectoryInfo struct {
	Message
	Flags    []string
	Name     string
	ReadOnly bool

	Exists, Recent, Unseen int
}

type DirectoryContents struct {
	Message
	Uids []uint32
}

type MessageInfo struct {
	Message
	BodyStructure *imap.BodyStructure
	Envelope      *imap.Envelope
	Flags         []string
	InternalDate  time.Time
	Size          uint32
	Uid           uint32
}

type FullMessage struct {
	Message
	Reader io.Reader
	Uid    uint32
}

type MessageBodyPart struct {
	Message
	Reader io.Reader
	Uid    uint32
}

type MessagesDeleted struct {
	Message
	Uids []uint32
}
