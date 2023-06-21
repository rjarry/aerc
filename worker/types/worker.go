package types

import (
	"container/list"
	"sync"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
)

type WorkerInteractor interface {
	log.Logger
	Actions() chan WorkerMessage
	ProcessAction(WorkerMessage) WorkerMessage
	PostAction(WorkerMessage, func(msg WorkerMessage))
	PostMessage(WorkerMessage, func(msg WorkerMessage))
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
	messageCallbacks map[int64]func(msg WorkerMessage)
	actionQueue      *list.List
	status           int32
	name             string

	sync.Mutex
	log.Logger
}

func NewWorker(name string) *Worker {
	return &Worker{
		Logger:           log.NewLogger(name, 2),
		actions:          make(chan WorkerMessage),
		actionCallbacks:  make(map[int64]func(msg WorkerMessage)),
		messageCallbacks: make(map[int64]func(msg WorkerMessage)),
		actionQueue:      list.New(),
		name:             name,
	}
}

func (worker *Worker) Actions() chan WorkerMessage {
	return worker.actions
}

func (worker *Worker) setId(msg WorkerMessage) {
	id := atomic.AddInt64(&lastId, 1)
	msg.setId(id)
}

const (
	idle int32 = iota
	busy
)

// Add a new task to the action queue without blocking. Start processing the
// queue in the background if needed.
func (worker *Worker) queue(msg WorkerMessage) {
	worker.Lock()
	defer worker.Unlock()
	worker.actionQueue.PushBack(msg)
	if atomic.LoadInt32(&worker.status) == idle {
		atomic.StoreInt32(&worker.status, busy)
		go worker.processQueue()
	}
}

// Start processing the action queue and write all messages to the actions
// channel, one by one. Stop when the action queue is empty.
func (worker *Worker) processQueue() {
	defer log.PanicHandler()
	for {
		worker.Lock()
		e := worker.actionQueue.Front()
		if e == nil {
			atomic.StoreInt32(&worker.status, idle)
			worker.Unlock()
			return
		}
		msg := worker.actionQueue.Remove(e).(WorkerMessage)
		worker.Unlock()
		worker.actions <- msg
	}
}

// PostAction posts an action to the worker. This method should not be called
// from the same goroutine that the worker runs in or deadlocks may occur
func (worker *Worker) PostAction(msg WorkerMessage, cb func(msg WorkerMessage)) {
	worker.setId(msg)
	// write to actions channel without blocking
	worker.queue(msg)

	if cb != nil {
		worker.Lock()
		worker.actionCallbacks[msg.getId()] = cb
		worker.Unlock()
	}
}

var WorkerMessages = make(chan WorkerMessage, 50)

// PostMessage posts an message to the UI. This method should not be called
// from the same goroutine that the UI runs in or deadlocks may occur
func (worker *Worker) PostMessage(msg WorkerMessage,
	cb func(msg WorkerMessage),
) {
	worker.setId(msg)
	msg.setAccount(worker.name)

	WorkerMessages <- msg

	if cb != nil {
		worker.Lock()
		worker.messageCallbacks[msg.getId()] = cb
		worker.Unlock()
	}
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
