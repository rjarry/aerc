package types

import (
	"container/list"
	"sync"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
)

var lastId int64 = 1 // access via atomic

type Backend interface {
	Run()
	Capabilities() *models.Capabilities
	PathSeparator() string
}

type Worker struct {
	Backend Backend
	Actions chan WorkerMessage
	Name    string
	logger  log.Logger

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
		logger:           log.NewLogger(name, 3),
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
		worker.Actions <- msg
	}
}

// PostAction posts an action to the worker. This method should not be called
// from the same goroutine that the worker runs in or deadlocks may occur
func (worker *Worker) PostAction(msg WorkerMessage, cb func(msg WorkerMessage)) {
	worker.setId(msg)

	if resp := msg.InResponseTo(); resp != nil {
		worker.Tracef("PostAction %T:%T", msg, resp)
	} else {
		worker.Tracef("PostAction %T", msg)
	}
	// write to Actions channel without blocking
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
	msg.setAccount(worker.Name)

	if resp := msg.InResponseTo(); resp != nil {
		worker.Tracef("PostMessage %T:%T", msg, resp)
	} else {
		worker.Tracef("PostMessage %T", msg)
	}
	WorkerMessages <- msg

	if cb != nil {
		worker.Lock()
		worker.messageCallbacks[msg.getId()] = cb
		worker.Unlock()
	}
}

func (worker *Worker) ProcessMessage(msg WorkerMessage) WorkerMessage {
	if resp := msg.InResponseTo(); resp != nil {
		worker.Tracef("ProcessMessage %T(%d):%T(%d)", msg, msg.getId(), resp, resp.getId())
	} else {
		worker.Tracef("ProcessMessage %T(%d)", msg, msg.getId())
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
		worker.Tracef("ProcessAction %T(%d):%T(%d)", msg, msg.getId(), resp, resp.getId())
	} else {
		worker.Tracef("ProcessAction %T(%d)", msg, msg.getId())
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
			Flags:    models.SeenFlag,
			Uid:      uid,
			Error:    err,
		},
		Message: RespondTo(msg),
	}, nil)
}

func (worker *Worker) PathSeparator() string {
	return worker.Backend.PathSeparator()
}

func (worker *Worker) Tracef(message string, args ...interface{}) {
	worker.logger.Tracef(message, args...)
}

func (worker *Worker) Debugf(message string, args ...interface{}) {
	worker.logger.Debugf(message, args...)
}

func (worker *Worker) Infof(message string, args ...interface{}) {
	worker.logger.Infof(message, args...)
}

func (worker *Worker) Warnf(message string, args ...interface{}) {
	worker.logger.Warnf(message, args...)
}

func (worker *Worker) Errorf(message string, args ...interface{}) {
	worker.logger.Errorf(message, args...)
}
