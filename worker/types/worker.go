package types

import (
	"log"
)

type Backend interface {
	Run()
}

type Worker struct {
	Actions   chan WorkerMessage
	Backend   Backend
	Callbacks map[WorkerMessage]func(msg WorkerMessage)
	Messages  chan WorkerMessage
	Logger    *log.Logger
}

func (worker *Worker) PostAction(msg WorkerMessage,
	cb func(msg WorkerMessage)) {

	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("(ui)=> %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("(ui)=> %T\n", msg)
	}
	worker.Actions <- msg

	if cb != nil {
		worker.Callbacks[msg] = cb
	}
}

func (worker *Worker) PostMessage(msg WorkerMessage,
	cb func(msg WorkerMessage)) {

	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("->(ui) %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("->(ui) %T\n", msg)
	}
	worker.Messages <- msg

	if cb != nil {
		worker.Callbacks[msg] = cb
	}
}

func (worker *Worker) ProcessMessage(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("(ui)<= %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("(ui)<= %T\n", msg)
	}
	if cb, ok := worker.Callbacks[msg.InResponseTo()]; ok {
		cb(msg)
		delete(worker.Callbacks, msg)
	}
	return msg
}

func (worker *Worker) ProcessAction(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("<-(ui) %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("<-(ui) %T\n", msg)
	}
	if cb, ok := worker.Callbacks[msg.InResponseTo()]; ok {
		cb(msg)
		delete(worker.Callbacks, msg)
	}
	return msg
}
