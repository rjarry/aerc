package imap

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-imap"
)

var errIdleTimeout = fmt.Errorf("idle timeout")

// idler manages the idle mode of the imap server. Enter idle mode if there's
// no other task and leave idle mode when a new task arrives. Idle mode is only
// used when the client is ready and connected. After a connection loss, make
// sure that idling returns gracefully and the worker remains responsive.
type idler struct {
	client    *imapClient
	debouncer *time.Timer
	debounce  time.Duration
	timeout   time.Duration
	worker    types.WorkerInteractor
	stop      chan struct{}
	start     chan struct{}
	done      chan error
}

func newIdler(cfg imapConfig, w types.WorkerInteractor, startIdler chan struct{}) *idler {
	return &idler{
		debouncer: nil,
		debounce:  cfg.idle_debounce,
		timeout:   cfg.idle_timeout,
		worker:    w,
		stop:      make(chan struct{}),
		start:     startIdler,
		done:      make(chan error),
	}
}

func (i *idler) SetClient(c *imapClient) {
	i.client = c
}

func (i *idler) ready() bool {
	return (i.client != nil && i.client.State() == imap.SelectedState)
}

func (i *idler) Start() {
	if !i.ready() {
		return
	}

	select {
	case <-i.stop:
		// stop channel is nil (probably after a debounce), we don't
		// want to close it
	default:
		close(i.stop)
	}

	// create new stop channel
	i.stop = make(chan struct{})

	// clear done channel
	clearing := true
	for clearing {
		select {
		case <-i.done:
			continue
		default:
			clearing = false
		}
	}

	i.worker.Tracef("idler (start): start idle after debounce")
	i.debouncer = time.AfterFunc(i.debounce, func() {
		i.start <- struct{}{}
		i.worker.Tracef("idler (start): started")
	})
}

func (i *idler) Execute() {
	if !i.ready() {
		return
	}

	// we need to call client.Idle in a goroutine since it is blocking call
	// and we still want to receive messages
	go func() {
		defer log.PanicHandler()

		start := time.Now()
		err := i.client.Idle(i.stop, nil)
		if err != nil {
			i.worker.Errorf("idle returned error: %v", err)
		}
		i.worker.Tracef("idler (execute): idleing for %s", time.Since(start))

		i.done <- err
	}()
}

func (i *idler) Stop() error {
	if !i.ready() {
		return nil
	}

	select {
	case <-i.stop:
		i.worker.Debugf("idler (stop): idler already stopped?")
		return nil
	default:
		close(i.stop)
	}

	if i.debouncer != nil {
		if i.debouncer.Stop() {
			i.worker.Tracef("idler (stop): debounced")
			return nil
		}
	}

	select {
	case err := <-i.done:
		i.worker.Tracef("idler (stop): idle stopped: %v", err)
		return err
	case <-time.After(i.timeout):
		i.worker.Errorf("idler (stop): cannot stop idle (timeout)")
		return errIdleTimeout
	}
}
