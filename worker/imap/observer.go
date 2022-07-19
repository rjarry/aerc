package imap

import (
	"fmt"
	"math"
	"sync"
	"time"

	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-imap"
)

// observer monitors the loggedOut channel of the imap client. If the logout
// signal is received, the observer will emit a connection error to the ui in
// order to start the reconnect cycle.
type observer struct {
	sync.Mutex
	config        imapConfig
	client        *imapClient
	worker        *types.Worker
	done          chan struct{}
	autoReconnect bool
	retries       int
	running       bool
}

func newObserver(cfg imapConfig, w *types.Worker) *observer {
	return &observer{config: cfg, worker: w, done: make(chan struct{})}
}

func (o *observer) SetClient(c *imapClient) {
	o.Stop()
	o.Lock()
	o.client = c
	o.Unlock()
	o.Start()
	o.retries = 0
}

func (o *observer) SetAutoReconnect(auto bool) {
	o.autoReconnect = auto
}

func (o *observer) AutoReconnect() bool {
	return o.autoReconnect
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
		o.log("runs already")
		return
	}
	if o.client == nil {
		return
	}
	if o.EmitIfNotConnected() {
		return
	}
	go func() {
		select {
		case <-o.client.LoggedOut():
			o.log("<-logout")
			if o.autoReconnect {
				o.emit("logged out")
			} else {
				o.log("ignore logout (auto-reconnect off)")
			}
		case <-o.done:
			o.log("<-done")
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

func (o *observer) DelayedReconnect() error {
	if o.client == nil {
		return nil
	}
	var wait time.Duration
	var reterr error

	if o.retries > 0 {
		backoff := int(math.Pow(1.8, float64(o.retries)))
		var err error
		wait, err = time.ParseDuration(fmt.Sprintf("%ds", backoff))
		if err != nil {
			return err
		}
		if wait > o.config.reconnect_maxwait {
			wait = o.config.reconnect_maxwait
		}

		reterr = fmt.Errorf("reconnect in %v", wait)
	} else {
		reterr = fmt.Errorf("reconnect")
	}

	go func() {
		<-time.After(wait)
		o.emit(reterr.Error())
	}()

	o.retries++
	return reterr
}

func (o *observer) emit(errMsg string) {
	o.log("disconnect done->")
	o.worker.PostMessage(&types.Done{
		Message: types.RespondTo(&types.Disconnect{})}, nil)
	o.log("connection error->")
	o.worker.PostMessage(&types.ConnError{
		Error: fmt.Errorf(errMsg),
	}, nil)
}

func (o *observer) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logging.Debugf("observer (%p) [running:%t] %s", o, o.running, msg)
}
