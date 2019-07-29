package ui

import (
	"sync/atomic"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/config"
)

type UI struct {
	Content DrawableInteractive
	exit    atomic.Value // bool
	ctx     *Context
	screen  tcell.Screen

	tcEvents chan tcell.Event
	invalid  int32 // access via atomic
}

func Initialize(conf *config.AercConfig,
	content DrawableInteractiveBeeper) (*UI, error) {

	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	if err = screen.Init(); err != nil {
		return nil, err
	}

	screen.Clear()
	screen.HideCursor()
	if conf.Ui.MouseEnabled {
		screen.EnableMouse()
	}

	width, height := screen.Size()

	state := UI{
		Content: content,
		ctx:     NewContext(width, height, screen),
		screen:  screen,

		tcEvents: make(chan tcell.Event, 10),
	}

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
	content.OnBeep(screen.Beep)
	content.Focus(true)

	return &state, nil
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
			state.ctx = NewContext(width, height, state.screen)
			state.Content.Invalidate()
		}
		state.Content.Event(event)
		more = true
	default:
	}

	wasInvalid := atomic.SwapInt32(&state.invalid, 0)
	if wasInvalid != 0 {
		state.Content.Draw(state.ctx)
		state.screen.Show()
		more = true
	}

	return more
}
