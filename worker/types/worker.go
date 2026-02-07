package types

import (
	"context"
	"sync"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
)

type WorkerInteractor interface {
	log.Logger
	Actions() chan WorkerMessage
	ProcessAction(WorkerMessage) WorkerMessage
	PostAction(context.Context, WorkerMessage, func(msg WorkerMessage))
	PostMessage(WorkerMessage, func(msg WorkerMessage))
	Unwrap() WorkerInteractor
	Name() string
}

var lastId int64 = 1 // access via atomic

type Backend interface {
	Run()
	Capabilities() *models.Capabilities
	PathSeparator() string
}

type Worker struct {
	Backend Backend

	actions          chan WorkerMessage
	actionCallbacks  map[int64]func(msg WorkerMessage)
	messages         chan WorkerMessage
	messageCallbacks map[int64]func(msg WorkerMessage)
	name             string

	sync.Mutex
	log.Logger
}

func NewWorker(name string, messages chan WorkerMessage) *Worker {
	return &Worker{
		Logger:           log.NewLogger(name, 2),
		actions:          make(chan WorkerMessage, 32),
		actionCallbacks:  make(map[int64]func(msg WorkerMessage)),
		messages:         messages,
		messageCallbacks: make(map[int64]func(msg WorkerMessage)),
		name:             name,
	}
}

func (worker *Worker) Unwrap() WorkerInteractor {
	return nil
}

func (worker *Worker) Actions() chan WorkerMessage {
	return worker.actions
}

func (worker *Worker) Name() string {
	return worker.name
}

func (worker *Worker) setId(msg WorkerMessage) {
	id := atomic.AddInt64(&lastId, 1)
	msg.setId(id)
}

// PostAction posts an action to the worker. This method should not be called
// from the same goroutine that the worker runs in or deadlocks may occur.
// If ctx is non-nil, it will be attached to the message for cancellation.
func (worker *Worker) PostAction(
	ctx context.Context, msg WorkerMessage, cb func(msg WorkerMessage),
) {
	worker.setId(msg)
	if ctx != nil {
		msg.setContext(ctx)
	}

	if cb != nil {
		worker.Lock()
		worker.actionCallbacks[msg.getId()] = cb
		worker.Unlock()
	}

	worker.actions <- msg
}

// PostMessage posts an message to the UI. This method should not be called
// from the same goroutine that the UI runs in or deadlocks may occur
func (worker *Worker) PostMessage(msg WorkerMessage,
	cb func(msg WorkerMessage),
) {
	worker.setId(msg)
	msg.setAccount(worker.name)

	if cb != nil {
		worker.Lock()
		worker.messageCallbacks[msg.getId()] = cb
		worker.Unlock()
	}

	worker.messages <- msg
}

func (worker *Worker) ProcessMessage(msg WorkerMessage) WorkerMessage {
	if inResponseTo := msg.InResponseTo(); inResponseTo != nil {
		worker.Lock()
		f, ok := worker.actionCallbacks[inResponseTo.getId()]
		worker.Unlock()
		if ok {
			f(msg)
			switch msg.(type) {
			case *Cancelled, *Done:
				worker.Lock()
				delete(worker.actionCallbacks, inResponseTo.getId())
				worker.Unlock()
			}
		}
	}
	return msg
}

func (worker *Worker) ProcessAction(msg WorkerMessage) WorkerMessage {
	if inResponseTo := msg.InResponseTo(); inResponseTo != nil {
		worker.Lock()
		f, ok := worker.messageCallbacks[inResponseTo.getId()]
		worker.Unlock()
		if ok {
			f(msg)
			if _, ok := msg.(*Done); ok {
				worker.Lock()
				delete(worker.messageCallbacks, inResponseTo.getId())
				worker.Unlock()
			}
		}
	}
	return msg
}

func (worker *Worker) PathSeparator() string {
	return worker.Backend.PathSeparator()
}
