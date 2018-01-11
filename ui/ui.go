package ui

import (
	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
)

func Initialize(conf *config.AercConfig) (*UIState, error) {
	state := UIState{
		Config:       conf,
		InvalidPanes: InvalidateAll,

		tbEvents: make(chan tb.Event, 10),
	}
	if err := tb.Init(); err != nil {
		return nil, err
	}
	tb.SetInputMode(tb.InputEsc | tb.InputMouse)
	tb.SetOutputMode(tb.Output256)
	go (func() {
		for !state.Exit {
			state.tbEvents <- tb.PollEvent()
		}
	})()
	return &state, nil
}

func (state *UIState) Close() {
	tb.Close()
}

func (state *UIState) AddTab(tab AercTab) {
	tab.SetParent(state)
	state.Tabs = append(state.Tabs, tab)
}

func (state *UIState) Invalidate(what uint) {
	state.InvalidPanes |= what
}

func (state *UIState) InvalidateFrom(tab AercTab) {
	if state.Tabs[state.SelectedTab] == tab {
		state.Invalidate(InvalidateTabView)
	}
}

func (state *UIState) calcGeometries() {
	width, height := tb.Size()
	// TODO: more
	state.Panes.TabView = Geometry{
		Row:    0,
		Col:    0,
		Width:  width,
		Height: height,
	}
}

func (state *UIState) Tick() bool {
	select {
	case event := <-state.tbEvents:
		switch event.Type {
		case tb.EventKey:
			if event.Key == tb.KeyEsc {
				state.Exit = true
			}
		case tb.EventResize:
			state.Invalidate(InvalidateAll)
		}
	default:
		// no-op
		break
	}
	if state.InvalidPanes != 0 {
		invalid := state.InvalidPanes
		state.InvalidPanes = 0
		if invalid&InvalidateAll == InvalidateAll {
			tb.Clear(tb.ColorDefault, tb.ColorDefault)
			state.calcGeometries()
		}
		if invalid&InvalidateTabView != 0 {
			tab := state.Tabs[state.SelectedTab]
			tab.Render(state.Panes.TabView)
		}
		tb.Flush()
	}
	return true
}
