package ui

import (
	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
)

type UI struct {
	Exit    bool
	Content Drawable
	ctx     *Context

	interactive []Interactive

	tbEvents      chan tb.Event
	invalidations chan interface{}
}

func Initialize(conf *config.AercConfig, content Drawable) (*UI, error) {
	if err := tb.Init(); err != nil {
		return nil, err
	}
	width, height := tb.Size()
	state := UI{
		Content: content,
		ctx:     NewContext(width, height),

		tbEvents:      make(chan tb.Event, 10),
		invalidations: make(chan interface{}),
	}
	tb.SetInputMode(tb.InputEsc | tb.InputMouse)
	tb.SetOutputMode(tb.Output256)
	go (func() {
		for !state.Exit {
			state.tbEvents <- tb.PollEvent()
		}
	})()
	go (func() { state.invalidations <- nil })()
	content.OnInvalidate(func(_ Drawable) {
		go (func() { state.invalidations <- nil })()
	})
	return &state, nil
}

func (state *UI) Close() {
	tb.Close()
}

func (state *UI) Tick() bool {
	select {
	case event := <-state.tbEvents:
		switch event.Type {
		case tb.EventKey:
			if event.Key == tb.KeyEsc {
				state.Exit = true
			}
		case tb.EventResize:
			tb.Clear(tb.ColorDefault, tb.ColorDefault)
			state.ctx = NewContext(event.Width, event.Height)
			state.Content.Invalidate()
		}
		if state.interactive != nil {
			for _, i := range state.interactive {
				i.Event(event)
			}
		}
	case <-state.invalidations:
		state.Content.Draw(state.ctx)
		tb.Flush()
	default:
		return false
	}
	return true
}

func (state *UI) AddInteractive(i Interactive) {
	state.interactive = append(state.interactive, i)
}
