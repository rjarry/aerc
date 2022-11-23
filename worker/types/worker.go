package types

import (
	"container/list"
	"sync"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
)

var lastId int64 = 1 // access via atomic

type Backend interface {
	Run()
}

type Worker struct {
	Backend Backend
	Actions chan WorkerMessage
	Name    string

	actionCallbacks  map[int64]func(msg WorkerMessage)
	messageCallbacks map[int64]func(msg WorkerMessage)
	actionQueue      *list.List
	status           int32

	sync.Mutex
}

func NewWorker(name string) *Worker {
	return &Worker{
		Actions:          make(chan WorkerMessage),
		Name:             name,
		actionCallbacks:  make(map[int64]func(msg WorkerMessage)),
		messageCallbacks: make(map[int64]func(msg WorkerMessage)),
		actionQueue:      list.New(),
	}
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

// Start processing the action queue and write all messages to the Actions
// channel, one by one. Stop when the action queue is empty.
func (worker *Worker) processQueue() {
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
		worker.Actions <- msg
	}
}

// PostAction posts an action to the worker. This method should not be called
// from the same goroutine that the worker runs in or deadlocks may occur
func (worker *Worker) PostAction(msg WorkerMessage, cb func(msg WorkerMessage)) {
	worker.setId(msg)

	if resp := msg.InResponseTo(); resp != nil {
		logging.Tracef("PostAction %T:%T", msg, resp)
	} else {
		logging.Tracef("PostAction %T", msg)
	}
	// write to Actions channel without blocking
	worker.queue(msg)

	if cb != nil {
		worker.Lock()
		worker.actionCallbacks[msg.getId()] = cb
		worker.Unlock()
	}
}

// PostMessage posts an message to the UI. This method should not be called
// from the same goroutine that the UI runs in or deadlocks may occur
func (worker *Worker) PostMessage(msg WorkerMessage,
	cb func(msg WorkerMessage),
) {
	worker.setId(msg)
	msg.setAccount(worker.Name)

	if resp := msg.InResponseTo(); resp != nil {
		logging.Tracef("PostMessage %T:%T", msg, resp)
	} else {
		logging.Tracef("PostMessage %T", msg)
	}
	ui.MsgChannel <- msg

	if cb != nil {
		worker.Lock()
		worker.messageCallbacks[msg.getId()] = cb
		worker.Unlock()
	}
}

func (worker *Worker) ProcessMessage(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		logging.Tracef("ProcessMessage %T(%d):%T(%d)", msg, msg.getId(), resp, resp.getId())
	} else {
		logging.Tracef("ProcessMessage %T(%d)", msg, msg.getId())
	}
	if inResponseTo := msg.InResponseTo(); inResponseTo != nil {
		worker.Lock()
		f, ok := worker.actionCallbacks[inResponseTo.getId()]
		worker.Unlock()
		if ok {
			f(msg)
			if _, ok := msg.(*Done); ok {
				worker.Lock()
				delete(worker.actionCallbacks, inResponseTo.getId())
				worker.Unlock()
			}
		}
	}
	return msg
}

func (worker *Worker) ProcessAction(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		logging.Tracef("ProcessAction %T(%d):%T(%d)", msg, msg.getId(), resp, resp.getId())
	} else {
		logging.Tracef("ProcessAction %T(%d)", msg, msg.getId())
	}
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

// PostMessageInfoError posts a MessageInfo message to the worker when an
// error was encountered fetching the message header
func (worker *Worker) PostMessageInfoError(msg WorkerMessage, uid uint32, err error) {
	worker.PostMessage(&MessageInfo{
		Info: &models.MessageInfo{
			Envelope: &models.Envelope{},
			Flags:    []models.Flag{models.SeenFlag},
			Uid:      uid,
			Error:    err,
		},
		Message: RespondTo(msg),
	}, nil)
}
