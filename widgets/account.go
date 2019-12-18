package widgets

import (
	"errors"
	"fmt"
	"log"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/sort"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/worker"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

var _ ProvidesMessages = (*AccountView)(nil)

type AccountView struct {
	acct    *config.AccountConfig
	aerc    *Aerc
	conf    *config.AercConfig
	dirlist *DirectoryList
	grid    *ui.Grid
	host    TabHost
	logger  *log.Logger
	msglist *MessageList
	worker  *types.Worker
}

func NewAccountView(aerc *Aerc, conf *config.AercConfig, acct *config.AccountConfig,
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
			aerc:   aerc,
			grid:   grid,
			host:   host,
			logger: logger,
		}
	}

	dirlist := NewDirectoryList(acct, &conf.Ui, logger, worker)
	if conf.Ui.SidebarWidth > 0 {
		grid.AddChild(ui.NewBordered(dirlist, ui.BORDER_RIGHT))
	}

	msglist := NewMessageList(conf, logger, aerc)
	grid.AddChild(msglist).At(0, 1)

	view := &AccountView{
		acct:    acct,
		aerc:    aerc,
		conf:    conf,
		dirlist: dirlist,
		grid:    grid,
		host:    host,
		logger:  logger,
		msglist: msglist,
		worker:  worker,
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

func (acct *AccountView) MouseEvent(localX int, localY int, event tcell.Event) {
	acct.grid.MouseEvent(localX, localY, event)
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
			if dir == "" && len(dirs) > 0 {
				dir = dirs[0]
			}
			if dir != "" {
				acct.dirlist.Select(dir)
			}

			acct.msglist.SetInitDone()
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

func (acct *AccountView) Store() *lib.MessageStore {
	if acct.msglist == nil {
		return nil
	}
	return acct.msglist.Store()
}

func (acct *AccountView) SelectedAccount() *AccountView {
	return acct
}

func (acct *AccountView) SelectedMessage() (*models.MessageInfo, error) {
	if len(acct.msglist.Store().Uids()) == 0 {
		return nil, errors.New("no message selected")
	}
	msg := acct.msglist.Selected()
	if msg == nil {
		return nil, errors.New("message not loaded")
	}
	return msg, nil
}

func (acct *AccountView) MarkedMessages() ([]*models.MessageInfo, error) {
	store := acct.Store()
	return msgInfoFromUids(store, store.Marked())
}

func (acct *AccountView) SelectedMessagePart() *PartInfo {
	return nil
}

func (acct *AccountView) onMessage(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.Done:
		switch msg.InResponseTo().(type) {
		case *types.OpenDirectory:
			if store, ok := acct.dirlist.SelectedMsgStore(); ok {
				// If we've opened this dir before, we can re-render it from
				// memory while we wait for the update and the UI feels
				// snappier. If not, we'll unset the store and show the spinner
				// while we download the UID list.
				acct.msglist.SetStore(store)
			} else {
				acct.msglist.SetStore(nil)
			}
		case *types.CreateDirectory:
			acct.dirlist.UpdateList(nil)
		}
	case *types.DirectoryInfo:
		if store, ok := acct.dirlist.MsgStore(msg.Info.Name); ok {
			store.Update(msg)
		} else {
			store = lib.NewMessageStore(acct.worker, msg.Info,
				acct.getSortCriteria(),
				func(msg *models.MessageInfo) {
					acct.conf.Triggers.ExecNewEmail(acct.acct,
						acct.conf, msg)
				}, func() {
					if acct.conf.Ui.NewMessageBell {
						acct.host.Beep()
					}
				})
			acct.dirlist.SetMsgStore(msg.Info.Name, store)
			store.OnUpdate(func(_ *lib.MessageStore) {
				store.OnUpdate(nil)
				acct.msglist.SetStore(store)
			})
		}
	case *types.DirectoryContents:
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			store.Update(msg)
		}
	case *types.FullMessage:
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			store.Update(msg)
		}
	case *types.MessageInfo:
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			store.Update(msg)
		}
	case *types.MessagesDeleted:
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			store.Update(msg)
		}
	case *types.Error:
		acct.logger.Printf("%v", msg.Error)
		acct.host.SetStatus(fmt.Sprintf("%v", msg.Error)).
			Color(tcell.ColorDefault, tcell.ColorRed)
	}
}

func (acct *AccountView) getSortCriteria() []*types.SortCriterion {
	if len(acct.conf.Ui.Sort) == 0 {
		return nil
	}
	criteria, err := sort.GetSortCriteria(acct.conf.Ui.Sort)
	if err != nil {
		acct.aerc.PushError(" ui.sort: " + err.Error())
		return nil
	}
	return criteria
}
