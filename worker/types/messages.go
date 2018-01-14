package types

import (
	"git.sr.ht/~sircmpwn/aerc2/config"
)

type WorkerMessage interface {
	InResponseTo() WorkerMessage
}

type Message struct {
	inResponseTo WorkerMessage
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

// Commands

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

func RespondTo(msg WorkerMessage) Message {
	return Message{
		inResponseTo: msg,
	}
}

func (m Message) InResponseTo() WorkerMessage {
	return m.inResponseTo
}
