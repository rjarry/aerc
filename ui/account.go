package ui

import (
	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/worker"
)

type AccountTab struct {
	Config *config.AccountConfig
	Worker *worker.Worker
	Parent *UIState

	counter int
}

func NewAccountTab(conf *config.AccountConfig, work *worker.Worker) *AccountTab {
	return &AccountTab{
		Config: conf,
		Worker: work,
	}
}

func (acc *AccountTab) Name() string {
	return acc.Config.Name
}

func (acc *AccountTab) SetParent(parent *UIState) {
	acc.Parent = parent
}

func (acc *AccountTab) Render(at Geometry) {
	cell := tb.Cell{
		Fg: tb.ColorDefault,
		Bg: tb.ColorDefault,
	}
	TPrintf(&at, cell, "%s %d", acc.Name(), acc.counter)
	acc.counter++
	if acc.counter%10000 == 0 {
		acc.counter = 0
	}
	acc.Parent.InvalidateFrom(acc)
}
