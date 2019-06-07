package types

import (
	"io"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"

	"git.sr.ht/~sircmpwn/aerc/config"
)

type WorkerMessage interface {
	InResponseTo() WorkerMessage
	getId() int64
	setId(id int64)
}

type Message struct {
	inResponseTo WorkerMessage
	id           int64
}

func RespondTo(msg WorkerMessage) Message {
	return Message{
		inResponseTo: msg,
	}
}

func (m Message) InResponseTo() WorkerMessage {
	return m.inResponseTo
}

func (m Message) getId() int64 {
	return m.id
}

func (m *Message) setId(id int64) {
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
	Part []int
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
	RFC822Headers *mail.Header
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
