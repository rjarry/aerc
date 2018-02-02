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

// TODO: Figure out a nice way of merging Ack and Done
type Ack struct {
	Message
}

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

type ListDirectories struct {
	Message
}

// Messages

type Directory struct {
	Message
	Name *string
}

// Respond with an Ack to approve or Disconnect to reject
type ApproveCertificate struct {
	Message
	CertPool *x509.CertPool
}
