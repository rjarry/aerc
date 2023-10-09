package ui

import (
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
)

// Use unbuffered channels (always blocking unless somebody can read
// immediately) We are merely using this as a proxy to tcell screen internal
// event channel.
var Events = make(chan tcell.Event)

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
	screen  tcell.Screen
	popover *Popover
	dirty   uint32 // == 1 if render has been queued in Redraw channel
}

func Initialize(content DrawableInteractive) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}

	if err = screen.Init(); err != nil {
		return err
	}

	screen.Clear()
	screen.HideCursor()
	screen.EnablePaste()

	width, height := screen.Size()

	state.content = content
	state.screen = screen
	state.ctx = NewContext(width, height, state.screen, onPopover)

	Invalidate()
	if beeper, ok := content.(DrawableInteractiveBeeper); ok {
		beeper.OnBeep(screen.Beep)
	}
	content.Focus(true)

	go state.screen.ChannelEvents(Events, Quit)

	return nil
}

func onPopover(p *Popover) {
	state.popover = p
}

func Exit() {
	close(Quit)
}

func Close() {
	state.screen.Fini()
}

func Render() {
	if atomic.SwapUint32(&state.dirty, 0) != 0 {
		// reset popover for the next Draw
		state.popover = nil
		state.content.Draw(state.ctx)
		if state.popover != nil {
			// if the Draw resulted in a popover, draw it
			state.popover.Draw(state.ctx)
		}
		state.screen.Show()
	}
}

func EnableMouse() {
	state.screen.EnableMouse()
}

func HandleEvent(event tcell.Event) {
	if event, ok := event.(*tcell.EventResize); ok {
		state.screen.Clear()
		width, height := event.Size()
		state.ctx = NewContext(width, height, state.screen, onPopover)
		Invalidate()
	}
	// if we have a popover, and it can handle the event, it does so
	if state.popover == nil || !state.popover.Event(event) {
		// otherwise, we send the event to the main content
		state.content.Event(event)
	}
}
