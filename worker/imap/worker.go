package imap

import (
	"git.sr.ht/~sircmpwn/aerc2/worker/types"

	"fmt"
)

type IMAPWorker struct {
	messages chan types.WorkerMessage
	actions  chan types.WorkerMessage
}

func NewIMAPWorker() *IMAPWorker {
	return &IMAPWorker{
		messages: make(chan types.WorkerMessage, 50),
		actions:  make(chan types.WorkerMessage, 50),
	}
}

func (w *IMAPWorker) GetMessage() types.WorkerMessage {
	select {
	case msg := <-w.messages:
		return msg
	default:
		return nil
	}
}

func (w *IMAPWorker) PostAction(msg types.WorkerMessage) {
	w.actions <- msg
}

func (w *IMAPWorker) handleMessage(_msg types.WorkerMessage) {
	switch msg := _msg.(type) {
	case types.Ping:
		w.messages <- types.Ack{
			Message: types.RespondTo(msg),
		}
	default:
		w.messages <- types.Unsupported{
			Message: types.RespondTo(_msg),
		}
	}
}

func (w *IMAPWorker) Run() {
	// TODO: IMAP shit
	for {
		select {
		case msg := <-w.actions:
			fmt.Printf("<= %T\n", msg)
			w.handleMessage(msg)
		default:
			// no-op
		}
	}
}
