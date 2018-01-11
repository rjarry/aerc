package ui

import (
	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
)

func Initialize(conf *config.AercConfig) (*UIState, error) {
	state := UIState{
		InvalidPanes: InvalidateAll,
		Tabs:         make([]AercTab, len(conf.Accounts)),
	}
	// TODO: Initialize each tab to a mailbox tab
	if err := tb.Init(); err != nil {
		return nil, err
	}
	tb.SetInputMode(tb.InputEsc | tb.InputMouse)
	tb.SetOutputMode(tb.Output256)
	return &state, nil
}

func (state *UIState) Close() {
	tb.Close()
}

func (state *UIState) Invalidate(what uint) {
	state.InvalidPanes |= what
}

func (state *UIState) Tick() bool {
	switch e := tb.PollEvent(); e.Type {
	case tb.EventKey:
		if e.Key == tb.KeyEsc {
			state.Exit = true
		}
	case tb.EventResize:
		state.Invalidate(InvalidateAll)
	}
	if state.InvalidPanes != 0 {
		// TODO: re-render
		state.InvalidPanes = 0
	}
	return true
}
