package widgets

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/marker"
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
	sync.Mutex
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

	split         *MessageViewer
	splitSize     int
	splitDebounce *time.Timer
	splitMsg      *models.MessageInfo
	splitDir      string

	// Check-mail ticker
	ticker       *time.Ticker
	checkingMail bool
}

func (acct *AccountView) UiConfig() *config.UIConfig {
	if dirlist := acct.Directories(); dirlist != nil {
		return dirlist.UiConfig("")
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

	worker, err := worker.NewWorker(acct.Source, acct.Name)
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

func (acct *AccountView) PushWarning(warning string) {
	acct.aerc.PushWarning(fmt.Sprintf("%s: %s", acct.acct.Name, warning))
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

func (acct *AccountView) Invalidate() {
	ui.Invalidate()
}

func (acct *AccountView) Draw(ctx *ui.Context) {
	if acct.state.SetWidth(ctx.Width()) {
		acct.UpdateStatus()
	}
	if acct.SplitSize() > 0 {
		acct.UpdateSplitView()
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
	if store := acct.Store(); store != nil {
		return store.Marker().Marked(), nil
	}
	return nil, errors.New("no store available")
}

func (acct *AccountView) SelectedMessagePart() *PartInfo {
	return nil
}

func (acct *AccountView) isSelected() bool {
	return acct == acct.aerc.SelectedAccount()
}

func (acct *AccountView) onMessage(msg types.WorkerMessage) {
	msg = acct.worker.ProcessMessage(msg)
	switch msg := msg.(type) {
	case *types.Done:
		switch msg.InResponseTo().(type) {
		case *types.Connect, *types.Reconnect:
			acct.SetStatus(statusline.ConnectionActivity("Listing mailboxes..."))
			logging.Tracef("Listing mailboxes...")
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
				logging.Infof("[%s] connected.", acct.acct.Name)
				acct.SetStatus(statusline.SetConnected(true))
				acct.newConn = true
			})
		case *types.Disconnect:
			acct.dirlist.ClearList()
			acct.msglist.SetStore(nil)
			logging.Infof("[%s] disconnected.", acct.acct.Name)
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
			if acct.newConn {
				acct.checkMailOnStartup()
			}
		}
	case *types.DirectoryInfo:
		if store, ok := acct.dirlist.MsgStore(msg.Info.Name); ok {
			store.Update(msg)
		} else {
			name := msg.Info.Name
			store = lib.NewMessageStore(acct.worker, msg.Info,
				acct.GetSortCriteria(),
				acct.dirlist.UiConfig(name).ThreadingEnabled,
				acct.dirlist.UiConfig(name).ForceClientThreads,
				acct.dirlist.UiConfig(name).ClientThreadsDelay,
				acct.dirlist.UiConfig(name).ReverseOrder,
				acct.dirlist.UiConfig(name).ReverseThreadOrder,
				acct.dirlist.UiConfig(name).SortThreadSiblings,
				func(msg *models.MessageInfo) {
					acct.conf.Triggers.ExecNewEmail(acct.acct,
						acct.conf, msg)
				}, func() {
					if acct.dirlist.UiConfig(name).NewMessageBell {
						acct.host.Beep()
					}
				})
			store.SetMarker(marker.New(store))
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
		if acct.newConn && len(msg.Uids) == 0 {
			acct.checkMailOnStartup()
		}
	case *types.DirectoryThreaded:
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			if acct.msglist.Store() == nil {
				acct.msglist.SetStore(store)
			}
			store.Update(msg)
			acct.SetStatus(statusline.Threading(store.ThreadedView()))
		}
		if acct.newConn && len(msg.Threads) == 0 {
			acct.checkMailOnStartup()
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
		acct.updateDirCounts(msg.Destination, msg.Uids)
	case *types.MessagesMoved:
		acct.updateDirCounts(msg.Destination, msg.Uids)
	case *types.LabelList:
		acct.labels = msg.Labels
	case *types.ConnError:
		logging.Errorf("[%s] connection error: %v", acct.acct.Name, msg.Error)
		acct.SetStatus(statusline.SetConnected(false))
		acct.PushError(msg.Error)
		acct.msglist.SetStore(nil)
		acct.worker.PostAction(&types.Reconnect{}, nil)
	case *types.Error:
		logging.Errorf("[%s] unexpected error: %v", acct.acct.Name, msg.Error)
		acct.PushError(msg.Error)
	}
	acct.UpdateStatus()
}

func (acct *AccountView) updateDirCounts(destination string, uids []uint32) {
	// Only update the destination destStore if it is initialized
	if destStore, ok := acct.dirlist.MsgStore(destination); ok {
		var recent, unseen int
		var accurate bool = true
		for _, uid := range uids {
			// Get the message from the originating store
			msg, ok := acct.Store().Messages[uid]
			if !ok {
				continue
			}
			// If message that was not yet loaded is copied
			if msg == nil {
				accurate = false
				break
			}
			seen := false
			for _, flag := range msg.Flags {
				if flag == models.SeenFlag {
					seen = true
				}
				if flag == models.RecentFlag {
					recent++
				}
			}
			if !seen {
				unseen++
			}
		}
		if accurate {
			destStore.DirInfo.Recent += recent
			destStore.DirInfo.Unseen += unseen
			destStore.DirInfo.Exists += len(uids)
			// True. For imap, we don't have the message in the store until we
			// Select so we need to rely on the math we just did for accurate
			// counts
			destStore.DirInfo.AccurateCounts = true
		} else {
			destStore.DirInfo.Exists += len(uids)
			// False to trigger recount of recent/unseen
			destStore.DirInfo.AccurateCounts = false
		}
	}
}

func (acct *AccountView) GetSortCriteria() []*types.SortCriterion {
	if len(acct.UiConfig().Sort) == 0 {
		return nil
	}
	criteria, err := sort.GetSortCriteria(acct.UiConfig().Sort)
	if err != nil {
		acct.PushError(fmt.Errorf("ui sort: %w", err))
		return nil
	}
	return criteria
}

func (acct *AccountView) CheckMail() {
	acct.Lock()
	defer acct.Unlock()
	if acct.checkingMail {
		return
	}
	// Exclude selected mailbox, per IMAP specification
	exclude := append(acct.AccountConfig().CheckMailExclude, acct.dirlist.Selected()) //nolint:gocritic // intentional append to different slice
	dirs := acct.dirlist.List()
	dirs = acct.dirlist.FilterDirs(dirs, acct.AccountConfig().CheckMailInclude, false)
	dirs = acct.dirlist.FilterDirs(dirs, exclude, true)
	logging.Debugf("Checking for new mail on account %s", acct.Name())
	acct.SetStatus(statusline.ConnectionActivity("Checking for new mail..."))
	msg := &types.CheckMail{
		Directories: dirs,
		Command:     acct.acct.CheckMailCmd,
		Timeout:     acct.acct.CheckMailTimeout,
	}
	acct.checkingMail = true

	var cb func(types.WorkerMessage)
	cb = func(response types.WorkerMessage) {
		dirsMsg, ok := response.(*types.CheckMailDirectories)
		if ok {
			checkMailMsg := &types.CheckMail{
				Directories: dirsMsg.Directories,
				Command:     acct.acct.CheckMailCmd,
				Timeout:     acct.acct.CheckMailTimeout,
			}
			acct.worker.PostAction(checkMailMsg, cb)
		} else { // Done
			acct.SetStatus(statusline.ConnectionActivity(""))
			acct.Lock()
			acct.checkingMail = false
			acct.Unlock()
		}
	}
	acct.worker.PostAction(msg, cb)
}

// CheckMailReset resets the check-mail timer
func (acct *AccountView) CheckMailReset() {
	if acct.ticker != nil {
		d := acct.AccountConfig().CheckMail
		acct.ticker = time.NewTicker(d)
	}
}

func (acct *AccountView) checkMailOnStartup() {
	if acct.AccountConfig().CheckMail.Minutes() > 0 {
		acct.newConn = false
		acct.CheckMail()
	}
}

func (acct *AccountView) CheckMailTimer(d time.Duration) {
	acct.ticker = time.NewTicker(d)
	go func() {
		for range acct.ticker.C {
			if !acct.state.Connected() {
				continue
			}
			acct.CheckMail()
		}
	}()
}

func (acct *AccountView) clearSplit() {
	if acct.split != nil {
		acct.split.Close()
	}
	acct.splitSize = 0
	acct.splitDir = ""
	acct.split = nil
	acct.grid = ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return acct.UiConfig().SidebarWidth
		}},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	if acct.uiConf.SidebarWidth > 0 {
		acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.uiConf))
	}
	acct.grid.AddChild(acct.msglist).At(0, 1)
	ui.Invalidate()
}

func (acct *AccountView) UpdateSplitView() {
	if acct.Store() == nil {
		return
	}
	if acct.splitMsg == acct.msglist.Selected() {
		return
	}
	if acct.splitDebounce != nil {
		acct.splitDebounce.Stop()
	}
	fn := func() {
		msg, err := acct.SelectedMessage()
		if err != nil {
			return
		}
		lib.NewMessageStoreView(msg, false, acct.Store(), acct.aerc.Crypto, acct.aerc.DecryptKeys,
			func(view lib.MessageView, err error) {
				if err != nil {
					acct.aerc.PushError(err.Error())
					return
				}
				orig := acct.split
				acct.split = NewMessageViewer(acct, acct.conf, view)
				acct.grid.ReplaceChild(orig, acct.split)
				if orig != nil {
					orig.Close()
				}
			})
		acct.splitMsg = msg
		ui.Invalidate()
	}
	acct.splitDebounce = time.AfterFunc(100*time.Millisecond, func() {
		ui.QueueFunc(fn)
	})
}

func (acct *AccountView) SplitSize() int {
	return acct.splitSize
}

func (acct *AccountView) SplitDirection() string {
	return acct.splitDir
}

// Split splits the message list view horizontally. The message list will be n
// rows high. If n is 0, any existing split is removed
func (acct *AccountView) Split(n int) error {
	if n == 0 {
		acct.clearSplit()
		return nil
	}
	msg, err := acct.SelectedMessage()
	if err != nil {
		return fmt.Errorf("could not create split: %w", err)
	}
	acct.splitSize = n
	acct.splitDir = "split"
	if acct.split != nil {
		acct.split.Close()
	}
	acct.grid = ui.NewGrid().Rows([]ui.GridSpec{
		// Add 1 so that the splitSize is the number of visible messages
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(acct.splitSize + 1)},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return acct.UiConfig().SidebarWidth
		}},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	if acct.uiConf.SidebarWidth > 0 {
		acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.uiConf)).Span(2, 1)
	}
	acct.grid.AddChild(ui.NewBordered(acct.msglist, ui.BORDER_BOTTOM, acct.uiConf)).At(0, 1)
	lib.NewMessageStoreView(msg, false, acct.Store(), acct.aerc.Crypto, acct.aerc.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				acct.aerc.PushError(err.Error())
				return
			}
			acct.split = NewMessageViewer(acct, acct.conf, view)
			acct.grid.AddChild(acct.split).At(1, 1)
		})
	ui.Invalidate()
	return nil
}

// Vsplit splits the message list view vertically. The message list will be n
// rows wide. If n is 0, any existing split is removed
func (acct *AccountView) Vsplit(n int) error {
	if n == 0 {
		acct.clearSplit()
		return nil
	}
	msg, err := acct.SelectedMessage()
	if err != nil {
		return fmt.Errorf("could not create split: %w", err)
	}
	acct.splitSize = n
	acct.splitDir = "vsplit"
	if acct.split != nil {
		acct.split.Close()
	}
	acct.grid = ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return acct.UiConfig().SidebarWidth
		}},
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(acct.splitSize)},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	if acct.uiConf.SidebarWidth > 0 {
		acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.uiConf)).At(0, 0)
	}
	acct.grid.AddChild(ui.NewBordered(acct.msglist, ui.BORDER_RIGHT, acct.uiConf)).At(0, 1)
	lib.NewMessageStoreView(msg, false, acct.Store(), acct.aerc.Crypto, acct.aerc.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				acct.aerc.PushError(err.Error())
				return
			}
			acct.split = NewMessageViewer(acct, acct.conf, view)
			acct.grid.AddChild(acct.split).At(0, 2)
		})
	ui.Invalidate()
	return nil
}
