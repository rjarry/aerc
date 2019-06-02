package types

import (
	"log"
	"sync/atomic"
)

var lastId int64 = 1 // access via atomic

type Backend interface {
	Run()
}

type Worker struct {
	Backend  Backend
	Actions  chan WorkerMessage
	Messages chan WorkerMessage
	Logger   *log.Logger

	actionCallbacks  map[int64]func(msg WorkerMessage)
	messageCallbacks map[int64]func(msg WorkerMessage)
}

func NewWorker(logger *log.Logger) *Worker {
	return &Worker{
		Actions:          make(chan WorkerMessage, 50),
		Messages:         make(chan WorkerMessage, 50),
		Logger:           logger,
		actionCallbacks:  make(map[int64]func(msg WorkerMessage)),
		messageCallbacks: make(map[int64]func(msg WorkerMessage)),
	}
}

func (worker *Worker) setId(msg WorkerMessage) {
	id := atomic.AddInt64(&lastId, 1)
	msg.setId(id)
}

func (worker *Worker) PostAction(msg WorkerMessage,
	cb func(msg WorkerMessage)) {

	worker.setId(msg)

	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("(ui)=> %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("(ui)=> %T\n", msg)
	}
	worker.Actions <- msg

	if cb != nil {
		worker.actionCallbacks[msg.getId()] = cb
	}
}

func (worker *Worker) PostMessage(msg WorkerMessage,
	cb func(msg WorkerMessage)) {

	worker.setId(msg)

	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("->(ui) %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("->(ui) %T\n", msg)
	}
	worker.Messages <- msg

	if cb != nil {
		worker.messageCallbacks[msg.getId()] = cb
	}
}

func (worker *Worker) ProcessMessage(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("(ui)<= %T(%d):%T(%d)\n",
			msg, msg.getId(), resp, resp.getId())
	} else {
		worker.Logger.Printf("(ui)<= %T(%d)\n", msg, msg.getId())
	}
	if inResponseTo := msg.InResponseTo(); inResponseTo != nil {
		if f, ok := worker.actionCallbacks[inResponseTo.getId()]; ok {
			f(msg)
			if _, ok := msg.(*Done); ok {
				delete(worker.actionCallbacks, inResponseTo.getId())
			}
		}
	}
	return msg
}

func (worker *Worker) ProcessAction(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("<-(ui) %T(%d):%T(%d)\n",
			msg, msg.getId(), resp, resp.getId())
	} else {
		worker.Logger.Printf("<-(ui) %T(%d)\n", msg, msg.getId())
	}
	if inResponseTo := msg.InResponseTo(); inResponseTo != nil {
		if f, ok := worker.messageCallbacks[inResponseTo.getId()]; ok {
			f(msg)
			if _, ok := msg.(*Done); ok {
				delete(worker.messageCallbacks, inResponseTo.getId())
			}
		}
	}
	return msg
}
