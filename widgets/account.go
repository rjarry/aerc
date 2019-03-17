package widgets

import (
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type AccountView struct {
	acct         *config.AccountConfig
	conf         *config.AercConfig
	dirlist      *DirectoryList
	grid         *ui.Grid
	logger       *log.Logger
	interactive  []ui.Interactive
	onInvalidate func(d ui.Drawable)
	runCmd       func(cmd string) error
	msglist      *MessageList
	msgStores    map[string]*lib.MessageStore
	pendingKeys  []config.KeyStroke
	statusline   *StatusLine
	statusbar    *ui.Stack
	worker       *types.Worker
}

func NewAccountView(conf *config.AercConfig, acct *config.AccountConfig,
	logger *log.Logger, runCmd func(cmd string) error) *AccountView {

	statusbar := ui.NewStack()
	statusline := NewStatusLine()
	statusbar.Push(statusline)

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_EXACT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_EXACT, conf.Ui.SidebarWidth},
		{ui.SIZE_WEIGHT, 1},
	})
	grid.AddChild(statusbar).At(1, 1)

	worker, err := worker.NewWorker(acct.Source, logger)
	if err != nil {
		statusline.Set(fmt.Sprintf("%s", err))
		return &AccountView{
			acct:       acct,
			grid:       grid,
			logger:     logger,
			statusline: statusline,
		}
	}

	dirlist := NewDirectoryList(acct, logger, worker)
	grid.AddChild(ui.NewBordered(dirlist, ui.BORDER_RIGHT)).Span(2, 1)

	msglist := NewMessageList(logger)
	grid.AddChild(msglist).At(0, 1)

	view := &AccountView{
		acct:       acct,
		conf:       conf,
		dirlist:    dirlist,
		grid:       grid,
		logger:     logger,
		msglist:    msglist,
		msgStores:  make(map[string]*lib.MessageStore),
		runCmd:     runCmd,
		statusbar:  statusbar,
		statusline: statusline,
		worker:     worker,
	}

	go worker.Backend.Run()
	go func() {
		for {
			msg := <-worker.Messages
			msg = worker.ProcessMessage(msg)
			view.onMessage(msg)
		}
	}()

	worker.PostAction(&types.Configure{Config: acct}, nil)
	worker.PostAction(&types.Connect{}, view.connected)
	statusline.Set("Connecting...")

	return view
}

func (acct *AccountView) Name() string {
	return acct.acct.Name
}

func (acct *AccountView) Children() []ui.Drawable {
	return acct.grid.Children()
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

func (acct *AccountView) popInteractive() {
	acct.interactive = acct.interactive[:len(acct.interactive)-1]
	if len(acct.interactive) != 0 {
		acct.interactive[len(acct.interactive)-1].Focus(true)
	}
}

func (acct *AccountView) pushInteractive(item ui.Interactive) {
	if len(acct.interactive) != 0 {
		acct.interactive[len(acct.interactive)-1].Focus(false)
	}
	acct.interactive = append(acct.interactive, item)
	item.Focus(true)
}

func (acct *AccountView) beginExCommand() {
	exline := NewExLine(func(command string) {
		err := acct.runCmd(command)
		if err != nil {
			acct.statusline.Push(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorRed, tcell.ColorWhite)
		}
		acct.statusbar.Pop()
		acct.popInteractive()
	}, func() {
		acct.statusbar.Pop()
		acct.popInteractive()
	})
	acct.pushInteractive(exline)
	acct.statusbar.Push(exline)
}

func (acct *AccountView) Event(event tcell.Event) bool {
	if len(acct.interactive) != 0 {
		return acct.interactive[len(acct.interactive)-1].Event(event)
	}

	switch event := event.(type) {
	case *tcell.EventKey:
		acct.pendingKeys = append(acct.pendingKeys, config.KeyStroke{
			Key:  event.Key(),
			Rune: event.Rune(),
		})
		result, output := acct.conf.Lbinds.GetBinding(acct.pendingKeys)
		switch result {
		case config.BINDING_FOUND:
			acct.pendingKeys = []config.KeyStroke{}
			for _, stroke := range output {
				simulated := tcell.NewEventKey(
					stroke.Key, stroke.Rune, tcell.ModNone)
				acct.Event(simulated)
			}
		case config.BINDING_INCOMPLETE:
			return false
		case config.BINDING_NOT_FOUND:
			acct.pendingKeys = []config.KeyStroke{}
			if event.Rune() == ':' {
				acct.beginExCommand()
				return true
			}
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
				if _dir == acct.acct.Default {
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
	}
}

func (acct *AccountView) Directories() *DirectoryList {
	return acct.dirlist
}

func (acct *AccountView) Messages() *MessageList {
	return acct.msglist
}

func (acct *AccountView) onMessage(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.Done:
		switch msg.InResponseTo().(type) {
		case *types.OpenDirectory:
			if store, ok := acct.msgStores[acct.dirlist.selected]; ok {
				// If we've opened this dir before, we can re-render it from
				// memory while we wait for the update and the UI feels
				// snappier. If not, we'll unset the store and show the spinner
				// while we download the UID list.
				acct.msglist.SetStore(store)
			} else {
				acct.msglist.SetStore(nil)
			}
			acct.worker.PostAction(&types.FetchDirectoryContents{},
				func(msg types.WorkerMessage) {
					store := acct.msgStores[acct.dirlist.selected]
					acct.msglist.SetStore(store)
				})
		}
	case *types.DirectoryInfo:
		if store, ok := acct.msgStores[msg.Name]; ok {
			store.Update(msg)
		} else {
			acct.msgStores[msg.Name] = lib.NewMessageStore(acct.worker, msg)
		}
	case *types.DirectoryContents:
		store := acct.msgStores[acct.dirlist.selected]
		store.Update(msg)
	case *types.MessageInfo:
		store := acct.msgStores[acct.dirlist.selected]
		store.Update(msg)
	case *types.Error:
		acct.logger.Printf("%v", msg.Error)
		acct.statusline.Set(fmt.Sprintf("%v", msg.Error)).
			Color(tcell.ColorRed, tcell.ColorDefault)
	}
}
