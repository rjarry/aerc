package ui

import (
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/gdamore/tcell/v2"
)

const (
	// nominal state, UI is up to date
	CLEAN int32 = iota
	// redraw is required but not explicitly requested
	DIRTY
	// redraw has been explicitly requested
	REDRAW_PENDING
)

var MsgChannel = make(chan AercMsg, 50)

type AercFuncMsg struct {
	Func func()
}

// State of the UI. Any value other than 0 means the UI is in a dirty state.
// This should only be accessed via atomic operations to maintain thread safety
var uiState int32

// QueueRedraw marks the UI as invalid and sends a nil message into the
// MsgChannel. Nothing will handle this message, but a redraw will occur
func QueueRedraw() {
	if atomic.SwapInt32(&uiState, REDRAW_PENDING) != REDRAW_PENDING {
		MsgChannel <- nil
	}
}

// QueueFunc queues a function to be called in the main goroutine. This can be
// used to prevent race conditions from delayed functions
func QueueFunc(fn func()) {
	MsgChannel <- &AercFuncMsg{Func: fn}
}

// Invalidate marks the entire UI as invalid. Invalidate can be called from any
// goroutine
func Invalidate() {
	atomic.StoreInt32(&uiState, DIRTY)
}

type UI struct {
	Content DrawableInteractive
	exit    atomic.Value // bool
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
	}
	state.ctx = NewContext(width, height, screen, state.onPopover)

	state.exit.Store(false)

	Invalidate()
	if beeper, ok := content.(DrawableInteractiveBeeper); ok {
		beeper.OnBeep(screen.Beep)
	}
	content.Focus(true)

	if root, ok := content.(RootDrawable); ok {
		root.Initialize(&state)
	}

	return &state, nil
}

func (state *UI) onPopover(p *Popover) {
	state.popover = p
}

func (state *UI) ShouldExit() bool {
	return state.exit.Load().(bool)
}

func (state *UI) Exit() {
	state.exit.Store(true)
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

func (state *UI) ChannelEvents() {
	go func() {
		defer log.PanicHandler()
		for {
			MsgChannel <- state.screen.PollEvent()
		}
	}()
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
