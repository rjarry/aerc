package ui

import (
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
)

type UI struct {
	Content DrawableInteractive
	exit    atomic.Value // bool
	ctx     *Context
	screen  tcell.Screen
	popover *Popover

	tcEvents chan tcell.Event
	invalid  int32 // access via atomic
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

	width, height := screen.Size()

	state := UI{
		Content: content,
		screen:  screen,

		tcEvents: make(chan tcell.Event, 10),
	}
	state.ctx = NewContext(width, height, screen, state.onPopover)

	state.exit.Store(false)
	go func() {
		for !state.ShouldExit() {
			state.tcEvents <- screen.PollEvent()
		}
	}()

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

func (state *UI) Tick() bool {
	more := false

	select {
	case event := <-state.tcEvents:
		switch event := event.(type) {
		case *tcell.EventResize:
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
		more = true
	default:
	}

	wasInvalid := atomic.SwapInt32(&state.invalid, 0)
	if wasInvalid != 0 {
		if state.popover != nil {
			// if the previous frame had a popover, rerender the entire display
			state.Content.Invalidate()
			atomic.StoreInt32(&state.invalid, 0)
		}
		// reset popover for the next Draw
		state.popover = nil
		state.Content.Draw(state.ctx)
		if state.popover != nil {
			// if the Draw resulted in a popover, draw it
			state.popover.Draw(state.ctx)
		}
		state.screen.Show()
		more = true
	}

	return more
}

func (state *UI) EnableMouse() {
	state.screen.EnableMouse()
}
