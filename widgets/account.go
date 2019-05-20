package widgets

import (
	"fmt"
	"log"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/worker"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type AccountView struct {
	acct      *config.AccountConfig
	conf      *config.AercConfig
	dirlist   *DirectoryList
	grid      *ui.Grid
	host      TabHost
	logger    *log.Logger
	msglist   *MessageList
	msgStores map[string]*lib.MessageStore
	worker    *types.Worker
}

func NewAccountView(conf *config.AercConfig, acct *config.AccountConfig,
	logger *log.Logger, host TabHost) *AccountView {

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_EXACT, conf.Ui.SidebarWidth},
		{ui.SIZE_WEIGHT, 1},
	})

	worker, err := worker.NewWorker(acct.Source, logger)
	if err != nil {
		host.SetStatus(fmt.Sprintf("%s: %s", acct.Name, err)).
			Color(tcell.ColorDefault, tcell.ColorRed)
		return &AccountView{
			acct:   acct,
			grid:   grid,
			host:   host,
			logger: logger,
		}
	}

	dirlist := NewDirectoryList(acct, logger, worker)
	grid.AddChild(ui.NewBordered(dirlist, ui.BORDER_RIGHT))

	msglist := NewMessageList(conf, logger)
	grid.AddChild(msglist).At(0, 1)

	view := &AccountView{
		acct:      acct,
		conf:      conf,
		dirlist:   dirlist,
		grid:      grid,
		host:      host,
		logger:    logger,
		msglist:   msglist,
		msgStores: make(map[string]*lib.MessageStore),
		worker:    worker,
	}

	go worker.Backend.Run()

	worker.PostAction(&types.Configure{Config: acct}, nil)
	worker.PostAction(&types.Connect{}, view.connected)
	host.SetStatus("Connecting...")

	return view
}

func (acct *AccountView) Tick() bool {
	if acct.worker == nil {
		return false
	}
	select {
	case msg := <-acct.worker.Messages:
		msg = acct.worker.ProcessMessage(msg)
		acct.onMessage(msg)
		return true
	default:
		return false
	}
}

func (acct *AccountView) AccountConfig() *config.AccountConfig {
	return acct.acct
}

func (acct *AccountView) Worker() *types.Worker {
	return acct.worker
}

func (acct *AccountView) Logger() *log.Logger {
	return acct.logger
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

func (acct *AccountView) Focus(focus bool) {
	// TODO: Unfocus children I guess
}

func (acct *AccountView) connected(msg types.WorkerMessage) {
	switch msg.(type) {
	case *types.Done:
		acct.host.SetStatus("Listing mailboxes...")
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
			acct.host.SetStatus("Connected.")
		})
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
		}
	case *types.DirectoryInfo:
		if store, ok := acct.msgStores[msg.Name]; ok {
			store.Update(msg)
		} else {
			store = lib.NewMessageStore(acct.worker, msg)
			acct.msgStores[msg.Name] = store
			store.OnUpdate(func(_ *lib.MessageStore) {
				store.OnUpdate(nil)
				acct.msglist.SetStore(store)
			})
		}
	case *types.DirectoryContents:
		store := acct.msgStores[acct.dirlist.selected]
		store.Update(msg)
	case *types.FullMessage:
		store := acct.msgStores[acct.dirlist.selected]
		store.Update(msg)
	case *types.MessageInfo:
		store := acct.msgStores[acct.dirlist.selected]
		store.Update(msg)
	case *types.MessagesDeleted:
		store := acct.msgStores[acct.dirlist.selected]
		store.Update(msg)
	case *types.Error:
		acct.logger.Printf("%v", msg.Error)
		acct.host.SetStatus(fmt.Sprintf("%v", msg.Error)).
			Color(tcell.ColorDefault, tcell.ColorRed)
	}
}
