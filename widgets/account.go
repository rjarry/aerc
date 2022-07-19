package widgets

import (
	"errors"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/sort"
	"git.sr.ht/~rjarry/aerc/lib/statusline"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

var _ ProvidesMessages = (*AccountView)(nil)

type AccountView struct {
	acct    *config.AccountConfig
	aerc    *Aerc
	conf    *config.AercConfig
	dirlist DirectoryLister
	labels  []string
	grid    *ui.Grid
	host    TabHost
	msglist *MessageList
	worker  *types.Worker
	state   *statusline.State
	newConn bool // True if this is a first run after a new connection/reconnection
	uiConf  *config.UIConfig
}

func (acct *AccountView) UiConfig() *config.UIConfig {
	if dirlist := acct.Directories(); dirlist != nil {
		return dirlist.UiConfig()
	}
	return acct.uiConf
}

func NewAccountView(aerc *Aerc, conf *config.AercConfig, acct *config.AccountConfig,
	host TabHost, deferLoop chan struct{},
) (*AccountView, error) {
	acctUiConf := conf.GetUiConfig(map[config.ContextType]string{
		config.UI_CONTEXT_ACCOUNT: acct.Name,
	})

	view := &AccountView{
		acct:   acct,
		aerc:   aerc,
		conf:   conf,
		host:   host,
		state:  statusline.NewState(acct.Name, len(conf.Accounts) > 1, conf.Statusline),
		uiConf: acctUiConf,
	}

	view.grid = ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return view.UiConfig().SidebarWidth
		}},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	worker, err := worker.NewWorker(acct.Source)
	if err != nil {
		host.SetError(fmt.Sprintf("%s: %s", acct.Name, err))
		logging.Errorf("%s: %v", acct.Name, err)
		return view, err
	}
	view.worker = worker

	view.dirlist = NewDirectoryList(conf, acct, worker)
	if acctUiConf.SidebarWidth > 0 {
		view.grid.AddChild(ui.NewBordered(view.dirlist, ui.BORDER_RIGHT, acctUiConf))
	}

	view.msglist = NewMessageList(conf, aerc)
	view.grid.AddChild(view.msglist).At(0, 1)

	go func() {
		defer logging.PanicHandler()

		if deferLoop != nil {
			<-deferLoop
		}

		worker.Backend.Run()
	}()

	worker.PostAction(&types.Configure{Config: acct}, nil)
	worker.PostAction(&types.Connect{}, nil)
	view.SetStatus(statusline.ConnectionActivity("Connecting..."))
	if acct.CheckMail.Minutes() > 0 {
		view.CheckMailTimer(acct.CheckMail)
	}

	return view, nil
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

func (acct *AccountView) SetStatus(setters ...statusline.SetStateFunc) {
	for _, fn := range setters {
		fn(acct.state, acct.SelectedDirectory())
	}
	acct.UpdateStatus()
}

func (acct *AccountView) UpdateStatus() {
	if acct.isSelected() {
		acct.host.SetStatus(acct.state.StatusLine(acct.SelectedDirectory()))
	}
}

func (acct *AccountView) PushStatus(status string, expiry time.Duration) {
	acct.aerc.PushStatus(fmt.Sprintf("%s: %s", acct.acct.Name, status), expiry)
}

func (acct *AccountView) PushError(err error) {
	acct.aerc.PushError(fmt.Sprintf("%s: %v", acct.acct.Name, err))
}

func (acct *AccountView) AccountConfig() *config.AccountConfig {
	return acct.acct
}

func (acct *AccountView) Worker() *types.Worker {
	return acct.worker
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
	if acct.state.SetWidth(ctx.Width()) {
		acct.UpdateStatus()
	}
	acct.grid.Draw(ctx)
}

func (acct *AccountView) MouseEvent(localX int, localY int, event tcell.Event) {
	acct.grid.MouseEvent(localX, localY, event)
}

func (acct *AccountView) Focus(focus bool) {
	// TODO: Unfocus children I guess
}

func (acct *AccountView) Directories() DirectoryLister {
	return acct.dirlist
}

func (acct *AccountView) Labels() []string {
	return acct.labels
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

func (acct *AccountView) SelectedDirectory() string {
	return acct.dirlist.Selected()
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

func (acct *AccountView) MarkedMessages() ([]uint32, error) {
	store := acct.Store()
	return store.Marked(), nil
}

func (acct *AccountView) SelectedMessagePart() *PartInfo {
	return nil
}

func (acct *AccountView) isSelected() bool {
	return acct == acct.aerc.SelectedAccount()
}

func (acct *AccountView) onMessage(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.Done:
		switch msg.InResponseTo().(type) {
		case *types.Connect, *types.Reconnect:
			acct.SetStatus(statusline.ConnectionActivity("Listing mailboxes..."))
			logging.Debugf("Listing mailboxes...")
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
				logging.Infof("%s connected.", acct.acct.Name)
				acct.SetStatus(statusline.SetConnected(true))
				acct.newConn = true
			})
		case *types.Disconnect:
			acct.dirlist.ClearList()
			acct.msglist.SetStore(nil)
			logging.Infof("%s disconnected.", acct.acct.Name)
			acct.SetStatus(statusline.SetConnected(false))
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
		case *types.RemoveDirectory:
			acct.dirlist.UpdateList(nil)
		case *types.FetchMessageHeaders:
			if acct.newConn && acct.AccountConfig().CheckMail.Minutes() > 0 {
				acct.newConn = false
				acct.CheckMail()
			}
		}
	case *types.DirectoryInfo:
		if store, ok := acct.dirlist.MsgStore(msg.Info.Name); ok {
			store.Update(msg)
		} else {
			store = lib.NewMessageStore(acct.worker, msg.Info,
				acct.GetSortCriteria(),
				acct.UiConfig().ThreadingEnabled,
				acct.UiConfig().ForceClientThreads,
				func(msg *models.MessageInfo) {
					acct.conf.Triggers.ExecNewEmail(acct.acct,
						acct.conf, msg)
				}, func() {
					if acct.UiConfig().NewMessageBell {
						acct.host.Beep()
					}
				})
			acct.dirlist.SetMsgStore(msg.Info.Name, store)
		}
	case *types.DirectoryContents:
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			if acct.msglist.Store() == nil {
				acct.msglist.SetStore(store)
			}
			store.Update(msg)
			acct.SetStatus(statusline.Threading(store.ThreadedView()))
		}
	case *types.DirectoryThreaded:
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			if acct.msglist.Store() == nil {
				acct.msglist.SetStore(store)
			}
			store.Update(msg)
			acct.SetStatus(statusline.Threading(store.ThreadedView()))
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
			store.DirInfo.Exists -= len(msg.Uids)
			// False to trigger recount of recent/unseen
			store.DirInfo.AccurateCounts = false
			store.Update(msg)
		}
	case *types.MessagesCopied:
		// Only update the destination destStore if it is initialized
		if destStore, ok := acct.dirlist.MsgStore(msg.Destination); ok {
			var recent, unseen int
			for _, uid := range msg.Uids {
				// Get the message from the originating store
				msg, ok := acct.Store().Messages[uid]
				if !ok {
					continue
				}
				seen := false
				for _, flag := range msg.Flags {
					if flag == models.SeenFlag {
						seen = true
					}
					if flag == models.RecentFlag {
						recent = recent + 1
					}
				}
				if !seen {
					unseen = unseen + 1
				}
			}
			destStore.DirInfo.Recent += recent
			destStore.DirInfo.Unseen += unseen
			destStore.DirInfo.Exists += len(msg.Uids)
			// True. For imap, we don't have the message in the store until we
			// Select so we need to rely on the math we just did for accurate
			// counts
			destStore.DirInfo.AccurateCounts = true
		}
	case *types.LabelList:
		acct.labels = msg.Labels
	case *types.ConnError:
		logging.Errorf("%s connection error: %v", acct.acct.Name, msg.Error)
		acct.SetStatus(statusline.SetConnected(false))
		acct.PushError(msg.Error)
		acct.msglist.SetStore(nil)
		acct.worker.PostAction(&types.Reconnect{}, nil)
	case *types.Error:
		logging.Errorf("%s unexpected error: %v", acct.acct.Name, msg.Error)
		acct.PushError(msg.Error)
	}
	acct.UpdateStatus()
}

func (acct *AccountView) GetSortCriteria() []*types.SortCriterion {
	if len(acct.UiConfig().Sort) == 0 {
		return nil
	}
	criteria, err := sort.GetSortCriteria(acct.UiConfig().Sort)
	if err != nil {
		acct.PushError(fmt.Errorf("ui sort: %v", err))
		return nil
	}
	return criteria
}

func (acct *AccountView) CheckMail() {
	// Exclude selected mailbox, per IMAP specification
	exclude := append(acct.AccountConfig().CheckMailExclude, acct.dirlist.Selected())
	dirs := acct.dirlist.List()
	dirs = acct.dirlist.FilterDirs(dirs, acct.AccountConfig().CheckMailInclude, false)
	dirs = acct.dirlist.FilterDirs(dirs, exclude, true)
	logging.Infof("Checking for new mail on account %s", acct.Name())
	acct.SetStatus(statusline.ConnectionActivity("Checking for new mail..."))
	msg := &types.CheckMail{
		Directories: dirs,
		Command:     acct.acct.CheckMailCmd,
		Timeout:     acct.acct.CheckMailTimeout,
	}
	acct.worker.PostAction(msg, func(_ types.WorkerMessage) {
		acct.SetStatus(statusline.ConnectionActivity(""))
	})
}

func (acct *AccountView) CheckMailTimer(d time.Duration) {
	ticker := time.NewTicker(d)
	go func() {
		for range ticker.C {
			if !acct.state.Connected() {
				continue
			}
			acct.CheckMail()
		}
	}()
}
