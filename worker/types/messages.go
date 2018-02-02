package types

import (
	"crypto/x509"

	"git.sr.ht/~sircmpwn/aerc2/config"
)

type WorkerMessage interface {
	InResponseTo() WorkerMessage
}

type Message struct {
	inResponseTo WorkerMessage
}

func RespondTo(msg WorkerMessage) Message {
	return Message{
		inResponseTo: msg,
	}
}

func (m Message) InResponseTo() WorkerMessage {
	return m.inResponseTo
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
