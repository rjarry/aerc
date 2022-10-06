package ui

import (
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
)

var MsgChannel = make(chan AercMsg, 50)

// QueueRedraw sends a nil message into the MsgChannel. Nothing will handle this
// message, but a redraw will occur if the UI is marked as invalid
func QueueRedraw() {
	MsgChannel <- nil
}

type UI struct {
	Content DrawableInteractive
	exit    atomic.Value // bool
	ctx     *Context
	screen  tcell.Screen
	popover *Popover
	invalid int32 // access via atomic
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

	state.invalid = 1
	content.OnInvalidate(func(_ Drawable) {
		atomic.StoreInt32(&state.invalid, 1)
	})
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
	wasInvalid := atomic.SwapInt32(&state.invalid, 0)
	if wasInvalid != 0 {
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
		state.Content.Invalidate()
	}
	// if we have a popover, and it can handle the event, it does so
	if state.popover == nil || !state.popover.Event(event) {
		// otherwise, we send the event to the main content
		state.Content.Event(event)
	}
}
