package imap

import (
	"fmt"
	"time"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
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

func (w *IMAPWorker) handleMessage(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case types.Ping:
		w.messages <- types.Ack{
			Message: types.RespondTo(msg),
		}
	default:
		w.messages <- types.Unsupported{
			Message: types.RespondTo(msg),
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
			time.Sleep(100 * time.Millisecond)
		}
	}
}
