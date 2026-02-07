package imap

import (
	"fmt"
	"sync"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-imap"
)

// observer monitors the loggedOut channel of the imap client. If the logout
// signal is received, the observer will emit a connection error to the ui in
// order to start the reconnect cycle.
type observer struct {
	sync.Mutex
	client  *imapClient
	worker  types.WorkerInteractor
	done    chan struct{}
	running bool
}

func newObserver(w types.WorkerInteractor) *observer {
	return &observer{worker: w, done: make(chan struct{})}
}

func (o *observer) SetClient(c *imapClient) {
	o.Stop()
	o.Lock()
	o.client = c
	o.Unlock()
	o.Start()
}

func (o *observer) isClientConnected() bool {
	o.Lock()
	defer o.Unlock()
	return o.client != nil && o.client.State() == imap.SelectedState
}

func (o *observer) EmitIfNotConnected() bool {
	if !o.isClientConnected() {
		o.emit("imap client not connected: attempt reconnect")
		return true
	}
	return false
}

func (o *observer) IsRunning() bool {
	return o.running
}

func (o *observer) Start() {
	if o.running {
		return
	}
	if o.client == nil {
		return
	}
	if o.EmitIfNotConnected() {
		return
	}
	go func() {
		defer log.PanicHandler()
		select {
		case <-o.client.LoggedOut():
			o.emit("logged out")
		case <-o.done:
			break
		}
		o.running = false
		o.log("stopped")
	}()
	o.running = true
	o.log("started")
}

func (o *observer) Stop() {
	if o.client == nil {
		return
	}
	if o.done != nil {
		close(o.done)
	}
	o.done = make(chan struct{})
	o.running = false
}

func (o *observer) emit(errMsg string) {
	o.worker.PostMessage(&types.ConnError{
		Error: fmt.Errorf("%s", errMsg),
	}, nil)
}

func (o *observer) log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	o.worker.Tracef("observer (%p) [running:%t] %s", o, o.running, msg)
}
