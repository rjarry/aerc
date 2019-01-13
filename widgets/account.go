package widgets

import (
	"log"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type AccountView struct {
	conf         *config.AccountConfig
	grid         *ui.Grid
	logger       *log.Logger
	onInvalidate func(d ui.Drawable)
	worker       *types.Worker
}

func NewAccountView(conf *config.AccountConfig,
	logger *log.Logger, statusbar ui.Drawable) (*AccountView, error) {

	worker, err := worker.NewWorker(conf.Source, logger)
	if err != nil {
		return nil, err
	}

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_EXACT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_EXACT, 20},
		{ui.SIZE_WEIGHT, 1},
	})
	grid.AddChild(ui.NewBordered(
		ui.NewFill('s'), ui.BORDER_RIGHT)).Span(2, 1)
	grid.AddChild(ui.NewFill('.')).At(0, 1)
	grid.AddChild(statusbar).At(1, 1)

	acct := &AccountView{
		conf:   conf,
		grid:   grid,
		logger: logger,
		worker: worker,
	}

	go worker.Backend.Run()
	go func() {
		for {
			msg := <-worker.Messages
			msg = worker.ProcessMessage(msg)
			// TODO: dispatch to appropriate handlers
		}
	}()

	worker.PostAction(&types.Configure{Config: conf}, nil)
	worker.PostAction(&types.Connect{}, acct.connected)

	return acct, nil
}

func (acct *AccountView) connected(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.Done:
		acct.logger.Println("Connected.")
		acct.worker.PostAction(&types.ListDirectories{}, nil)
	case *types.CertificateApprovalRequest:
		// TODO: Ask the user
		acct.logger.Println("Approving certificate")
		acct.worker.PostAction(&types.ApproveCertificate{
			Message:  types.RespondTo(msg),
			Approved: true,
		}, acct.connected)
	default:
		acct.logger.Println("Connection failed.")
	}
}

func (acct *AccountView) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	acct.grid.OnInvalidate(onInvalidate)
}

func (acct *AccountView) Invalidate() {
	acct.grid.Invalidate()
}

func (acct *AccountView) Draw(ctx *ui.Context) {
	acct.grid.Draw(ctx)
}
