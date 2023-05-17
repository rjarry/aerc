package ui

import (
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
)

const (
	// nominal state, UI is up to date
	CLEAN int32 = iota
	// UI render has been queued in Redraw channel
	DIRTY
)

// State of the UI. Any value other than 0 means the UI is in a dirty state.
// This should only be accessed via atomic operations to maintain thread safety
var uiState int32

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
	if atomic.SwapInt32(&uiState, DIRTY) != DIRTY {
		Redraw <- true
	}
}

type UI struct {
	Content DrawableInteractive
	Quit    chan struct{}
	Events  chan tcell.Event
	ctx     *Context
	screen  tcell.Screen
	popover *Popover
}

func Initialize(content DrawableInteractive) (*UI, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	if err = screen.Init(); err != nil {
		return nil, err
	}

	screen.Clear()
	screen.HideCursor()
	screen.EnablePaste()

	width, height := screen.Size()

	state := UI{
		Content: content,
		screen:  screen,
		// Use unbuffered channels (always blocking unless somebody can
		// read immediately) We are merely using this as a proxy to
		// tcell screen internal event channel.
		Events: make(chan tcell.Event),
		Quit:   make(chan struct{}),
	}
	state.ctx = NewContext(width, height, screen, state.onPopover)

	Invalidate()
	if beeper, ok := content.(DrawableInteractiveBeeper); ok {
		beeper.OnBeep(screen.Beep)
	}
	content.Focus(true)

	if root, ok := content.(RootDrawable); ok {
		root.Initialize(&state)
	}
	go state.screen.ChannelEvents(state.Events, state.Quit)

	return &state, nil
}

func (state *UI) onPopover(p *Popover) {
	state.popover = p
}

func (state *UI) Exit() {
	close(state.Quit)
}

func (state *UI) Close() {
	state.screen.Fini()
}

func (state *UI) Render() {
	if atomic.SwapInt32(&uiState, CLEAN) != CLEAN {
		// reset popover for the next Draw
		state.popover = nil
		state.Content.Draw(state.ctx)
		if state.popover != nil {
			// if the Draw resulted in a popover, draw it
			state.popover.Draw(state.ctx)
		}
		state.screen.Show()
	}
}

func (state *UI) EnableMouse() {
	state.screen.EnableMouse()
}

func (state *UI) HandleEvent(event tcell.Event) {
	if event, ok := event.(*tcell.EventResize); ok {
		state.screen.Clear()
		width, height := event.Size()
		state.ctx = NewContext(width, height, state.screen, state.onPopover)
		Invalidate()
	}
	// if we have a popover, and it can handle the event, it does so
	if state.popover == nil || !state.popover.Event(event) {
		// otherwise, we send the event to the main content
		state.Content.Event(event)
	}
}
