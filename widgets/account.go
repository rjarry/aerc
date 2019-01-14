package widgets

import (
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type AccountView struct {
	conf         *config.AccountConfig
	dirlist      *DirectoryList
	grid         *ui.Grid
	logger       *log.Logger
	interactive  ui.Interactive
	onInvalidate func(d ui.Drawable)
	statusline   *StatusLine
	statusbar    *ui.Stack
	worker       *types.Worker
}

func NewAccountView(
	conf *config.AccountConfig, logger *log.Logger) *AccountView {

	statusbar := ui.NewStack()
	statusline := NewStatusLine()
	statusbar.Push(statusline)

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_EXACT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_EXACT, 20},
		{ui.SIZE_WEIGHT, 1},
	})
	spinner := NewSpinner()
	spinner.Start()
	grid.AddChild(spinner).At(0, 1)
	grid.AddChild(statusbar).At(1, 1)

	worker, err := worker.NewWorker(conf.Source, logger)
	if err != nil {
		// TODO: Update status line with error
		return &AccountView{
			conf:       conf,
			grid:       grid,
			logger:     logger,
			statusline: statusline,
		}
	}

	dirlist := NewDirectoryList(conf, logger, worker)
	grid.AddChild(ui.NewBordered(dirlist, ui.BORDER_RIGHT)).Span(2, 1)

	acct := &AccountView{
		conf:       conf,
		dirlist:    dirlist,
		grid:       grid,
		logger:     logger,
		statusline: statusline,
		statusbar:  statusbar,
		worker:     worker,
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
	statusline.Set("Connecting...")

	return acct
}

func (acct *AccountView) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	acct.grid.OnInvalidate(func(_ ui.Drawable) {
		onInvalidate(acct)
	})
}

func (acct *AccountView) Invalidate() {
	acct.grid.Invalidate()
}

func (acct *AccountView) Draw(ctx *ui.Context) {
	acct.grid.Draw(ctx)
}

func (acct *AccountView) Event(event tcell.Event) bool {
	if acct.interactive != nil {
		return acct.interactive.Event(event)
	}
	switch event := event.(type) {
	case *tcell.EventKey:
		if event.Rune() == ':' {
			exline := NewExLine(func(command string) {
				acct.statusline.Push(
					fmt.Sprintf("TODO: execute %s", command), 3*time.Second)
				acct.statusbar.Pop()
				acct.interactive = nil
			}, func() {
				acct.statusbar.Pop()
				acct.interactive = nil
			})
			acct.interactive = exline
			acct.statusbar.Push(exline)
			return true
		}
	}
	return false
}

func (acct *AccountView) connected(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.Done:
		acct.statusline.Set("Listing mailboxes...")
		acct.logger.Println("Listing mailboxes...")
		acct.dirlist.UpdateList(func(dirs []string) {
			var dir string
			for _, _dir := range dirs {
				if _dir == "INBOX" {
					dir = _dir
					break
				}
			}
			if dir == "" {
				dir = dirs[0]
			}
			acct.dirlist.Select(dir)
			acct.logger.Println("Connected.")
			acct.statusline.Set("Connected.")
		})
	case *types.CertificateApprovalRequest:
		// TODO: Ask the user
		acct.worker.PostAction(&types.ApproveCertificate{
			Message:  types.RespondTo(msg),
			Approved: true,
		}, acct.connected)
	case *types.Error:
		acct.logger.Printf("%v", msg.Error)
		acct.statusline.Set(fmt.Sprintf("%v", msg.Error)).
			Color(tcell.ColorRed, tcell.ColorDefault)
	}
}
