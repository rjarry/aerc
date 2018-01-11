package ui

import (
	"fmt"

	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type AccountTab struct {
	Config *config.AccountConfig
	Worker worker.Worker
	Parent *UIState

	counter int
	log     []string
}

func NewAccountTab(conf *config.AccountConfig) (*AccountTab, error) {
	work, err := worker.NewWorker(conf.Source)
	if err != nil {
		return nil, err
	}
	go work.Run()
	work.PostAction(types.Configure{Config: conf})
	return &AccountTab{
		Config: conf,
		Worker: work,
	}, nil
}

func (acc *AccountTab) Name() string {
	return acc.Config.Name
}

func (acc *AccountTab) SetParent(parent *UIState) {
	acc.Parent = parent
}

func (acc *AccountTab) Render(at Geometry) {
	cell := tb.Cell{
		Ch: ' ',
		Fg: tb.ColorDefault,
		Bg: tb.ColorDefault,
	}
	TFill(at, cell)
	TPrintf(&at, cell, "%s %d\n", acc.Name(), acc.counter)
	for _, str := range acc.log {
		TPrintf(&at, cell, "%s\n", str)
	}
	acc.counter++
	if acc.counter%10000 == 0 {
		acc.counter = 0
	}
	acc.Parent.InvalidateFrom(acc)
}

func (acc *AccountTab) GetChannel() chan types.WorkerMessage {
	return acc.Worker.GetMessages()
}

func (acc *AccountTab) HandleMessage(msg types.WorkerMessage) {
	acc.log = append(acc.log, fmt.Sprintf("<- %T", msg))
}
