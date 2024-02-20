package ui

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rockorager/vaxis"
)

// Use unbuffered channels (always blocking unless somebody can read
// immediately) We are merely using this as a proxy to the internal vaxis event
// channel.
var Events = make(chan vaxis.Event)

var Quit = make(chan struct{})

var Callbacks = make(chan func(), 50)

// QueueFunc queues a function to be called in the main goroutine. This can be
// used to prevent race conditions from delayed functions
func QueueFunc(fn func()) {
	Callbacks <- fn
}

// Use a buffered channel of size 1 to avoid blocking callers of Invalidate()
var Redraw = make(chan bool, 1)

// Invalidate marks the entire UI as invalid and request a redraw as soon as
// possible. Invalidate can be called from any goroutine and will never block.
func Invalidate() {
	if atomic.SwapUint32(&state.dirty, 1) != 1 {
		Redraw <- true
	}
}

var state struct {
	content DrawableInteractive
	ctx     *Context
	vx      *vaxis.Vaxis
	popover *Popover
	dirty   uint32 // == 1 if render has been queued in Redraw channel
	// == 1 if suspend is pending
	suspending uint32
}

func Initialize(content DrawableInteractive) error {
	opts := vaxis.Options{
		DisableMouse: !config.Ui.MouseEnabled,
	}
	vx, err := vaxis.New(opts)
	if err != nil {
		return err
	}

	vx.Window().Clear()
	vx.HideCursor()

	state.content = content
	state.vx = vx
	state.ctx = NewContext(state.vx, onPopover)
	vx.SetTitle("aerc")

	Invalidate()
	if beeper, ok := content.(DrawableInteractiveBeeper); ok {
		beeper.OnBeep(vx.Bell)
	}
	content.Focus(true)

	go func() {
		defer log.PanicHandler()
		for event := range vx.Events() {
			Events <- event
		}
	}()

	return nil
}

func onPopover(p *Popover) {
	state.popover = p
}

func Exit() {
	close(Quit)
}

var SuspendQueue = make(chan bool, 1)

func QueueSuspend() {
	if atomic.SwapUint32(&state.suspending, 1) != 1 {
		SuspendQueue <- true
	}
}

func Suspend() error {
	var err error
	if atomic.SwapUint32(&state.suspending, 0) != 0 {
		err = state.vx.Suspend()
		if err == nil {
			sigcont := make(chan os.Signal, 1)
			signal.Notify(sigcont, syscall.SIGCONT)
			err = syscall.Kill(0, syscall.SIGTSTP)
			if err == nil {
				<-sigcont
			}
			signal.Reset(syscall.SIGCONT)
			err = state.vx.Resume()
			state.content.Draw(state.ctx)
			state.vx.Render()
		}
	}
	return err
}

func Close() {
	state.vx.Close()
}

func Render() {
	if atomic.SwapUint32(&state.dirty, 0) != 0 {
		state.vx.Window().Clear()
		// reset popover for the next Draw
		state.popover = nil
		state.vx.HideCursor()
		state.content.Draw(state.ctx)
		if state.popover != nil {
			// if the Draw resulted in a popover, draw it
			state.popover.Draw(state.ctx)
		}
		state.vx.Render()
	}
}

func HandleEvent(event vaxis.Event) {
	switch event := event.(type) {
	case vaxis.Resize:
		state.ctx = NewContext(state.vx, onPopover)
		Invalidate()
	case vaxis.Redraw:
		Invalidate()
	default:
		// We never care about num or caps lock. Remove them so it
		// doesn't interefere with key matching
		if key, ok := event.(vaxis.Key); ok {
			key.Modifiers &^= vaxis.ModCapsLock
			key.Modifiers &^= vaxis.ModNumLock
			event = key
		}
		// if we have a popover, and it can handle the event, it does so
		if state.popover == nil || !state.popover.Event(event) {
			// otherwise, we send the event to the main content
			state.content.Event(event)
		}
	}
}
