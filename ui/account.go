package ui

import (
	"log"

	tb "github.com/nsf/termbox-go"

	"github.com/davecgh/go-spew/spew"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type AccountTab struct {
	Config  *config.AccountConfig
	Worker  worker.Worker
	Parent  *UIState
	logger  *log.Logger
	counter int
}

func NewAccountTab(conf *config.AccountConfig,
	logger *log.Logger) (*AccountTab, error) {

	work, err := worker.NewWorker(conf.Source, logger)
	if err != nil {
		return nil, err
	}
	go work.Run()
	work.PostAction(types.Configure{Config: conf})
	work.PostAction(types.Connect{})
	return &AccountTab{
		Config: conf,
		Worker: work,
		logger: logger,
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
	switch msg.InResponseTo().(type) {
	case types.Configure:
		// Avoid printing passwords
		acc.logger.Printf("<- %T\n", msg)
	default:
		acc.logger.Printf("<- %s", spew.Sdump(msg))
	}
}
