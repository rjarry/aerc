package ui

import (
	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
)

type UI struct {
	Exit    bool
	Content DrawableInteractive
	ctx     *Context
	screen  tcell.Screen

	tcEvents      chan tcell.Event
	invalidations chan interface{}
}

func Initialize(conf *config.AercConfig,
	content DrawableInteractive) (*UI, error) {

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
		ctx:     NewContext(width, height, screen),
		screen:  screen,

		tcEvents:      make(chan tcell.Event, 10),
		invalidations: make(chan interface{}),
	}
	go (func() {
		for !state.Exit {
			state.tcEvents <- screen.PollEvent()
		}
	})()
	go (func() {
		state.invalidations <- nil
	})()
	content.OnInvalidate(func(_ Drawable) {
		go (func() {
			state.invalidations <- nil
		})()
	})
	content.Focus(true)
	return &state, nil
}

func (state *UI) Close() {
	state.screen.Fini()
}

func (state *UI) Tick() bool {
	select {
	case event := <-state.tcEvents:
		switch event := event.(type) {
		case *tcell.EventResize:
			state.screen.Clear()
			width, height := event.Size()
			state.ctx = NewContext(width, height, state.screen)
			state.Content.Invalidate()
		}
		state.Content.Event(event)
	case <-state.invalidations:
		for {
			// Flush any other pending invalidations
			select {
			case <-state.invalidations:
				break
			default:
				goto done
			}
		}
	done:
		state.Content.Draw(state.ctx)
		state.screen.Show()
	default:
		return false
	}
	return true
}
