package models

import (
	"io"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
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
	BodyStructure *imap.BodyStructure
	Envelope      *imap.Envelope
	Flags         []string
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
