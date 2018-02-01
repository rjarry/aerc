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

type Ack struct {
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

type Ping struct {
	Message
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

// Messages

// Respond with an Ack to approve or Disconnect to reject
type ApproveCertificate struct {
	Message
	CertPool *x509.CertPool
}
