package ui

import (
	"log"

	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type AccountTab struct {
	Config  *config.AccountConfig
	Worker  *types.Worker
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
	go work.Backend.Run()
	acc := &AccountTab{
		Config: conf,
		Worker: work,
		logger: logger,
	}
	acc.Worker.PostAction(types.Configure{Config: conf}, nil)
	acc.Worker.PostAction(types.Connect{}, func(msg types.WorkerMessage) {
		if _, ok := msg.(types.Ack); ok {
			acc.logger.Println("Connected.")
		} else {
			acc.logger.Println("Connection failed.")
		}
	})
	return acc, nil
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
	return acc.Worker.Messages
}

func (acc *AccountTab) HandleMessage(msg types.WorkerMessage) {
	msg = acc.Worker.ProcessMessage(msg)
	switch msg.(type) {
	case types.Ack:
		// no-op
	case types.ApproveCertificate:
		// TODO: Ask the user
		acc.logger.Println("Approving certificate")
		acc.Worker.PostAction(types.Ack{
			Message: types.RespondTo(msg),
		}, nil)
	default:
		acc.Worker.PostAction(types.Unsupported{
			Message: types.RespondTo(msg),
		}, nil)
	}
}
