package types

import (
	"log"
	"sync"
)

type Backend interface {
	Run()
}

type Worker struct {
	Backend  Backend
	Actions  chan WorkerMessage
	Messages chan WorkerMessage
	Logger   *log.Logger

	callbacks map[WorkerMessage]func(msg WorkerMessage) // protected by mutex
	mutex     sync.Mutex
}

func NewWorker(logger *log.Logger) *Worker {
	return &Worker{
		Actions:   make(chan WorkerMessage, 50),
		Messages:  make(chan WorkerMessage, 50),
		Logger:    logger,
		callbacks: make(map[WorkerMessage]func(msg WorkerMessage)),
	}
}

func (worker *Worker) setCallback(msg WorkerMessage,
	cb func(msg WorkerMessage)) {

	if cb != nil {
		worker.mutex.Lock()
		worker.callbacks[msg] = cb
		worker.mutex.Unlock()
	}
}

func (worker *Worker) getCallback(msg WorkerMessage) (func(msg WorkerMessage),
	bool) {

	worker.mutex.Lock()
	cb, ok := worker.callbacks[msg]
	worker.mutex.Unlock()

	return cb, ok
}

func (worker *Worker) PostAction(msg WorkerMessage,
	cb func(msg WorkerMessage)) {

	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("(ui)=> %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("(ui)=> %T\n", msg)
	}
	worker.Actions <- msg

	worker.setCallback(msg, cb)
}

func (worker *Worker) PostMessage(msg WorkerMessage,
	cb func(msg WorkerMessage)) {

	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("->(ui) %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("->(ui) %T\n", msg)
	}
	worker.Messages <- msg

	worker.setCallback(msg, cb)
}

func (worker *Worker) ProcessMessage(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("(ui)<= %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("(ui)<= %T\n", msg)
	}
	if cb, ok := worker.getCallback(msg.InResponseTo()); ok {
		cb(msg)
	}
	return msg
}

func (worker *Worker) ProcessAction(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		worker.Logger.Printf("<-(ui) %T:%T\n", msg, resp)
	} else {
		worker.Logger.Printf("<-(ui) %T\n", msg)
	}
	if cb, ok := worker.getCallback(msg.InResponseTo()); ok {
		cb(msg)
	}
	return msg
}
