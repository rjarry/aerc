package app

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/hooks"
	"git.sr.ht/~rjarry/aerc/lib/marker"
	"git.sr.ht/~rjarry/aerc/lib/sort"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

var _ ProvidesMessages = (*AccountView)(nil)

type AccountView struct {
	sync.Mutex
	acct    *config.AccountConfig
	dirlist DirectoryLister
	labels  []string
	grid    *ui.Grid
	tab     *ui.Tab
	msglist *MessageList
	worker  *types.Worker
	state   state.AccountState
	newConn bool // True if this is a first run after a new connection/reconnection
	uiConf  *config.UIConfig

	split         *MessageViewer
	splitSize     int
	splitDebounce *time.Timer
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

func NewAccountView(
	acct *config.AccountConfig, deferLoop chan struct{},
) (*AccountView, error) {
	acctUiConf := config.Ui.ForAccount(acct.Name)

	view := &AccountView{
		acct:   acct,
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
		SetError(fmt.Sprintf("%s: %s", acct.Name, err))
		log.Errorf("%s: %v", acct.Name, err)
		return view, err
	}
	view.worker = worker

	view.dirlist = NewDirectoryList(acct, worker)
	if acctUiConf.SidebarWidth > 0 {
		view.grid.AddChild(ui.NewBordered(view.dirlist, ui.BORDER_RIGHT, acctUiConf))
	}

	view.msglist = NewMessageList(view)
	view.grid.AddChild(view.msglist).At(0, 1)

	view.dirlist.OnVirtualNode(func() {
		view.msglist.SetStore(nil)
		view.Invalidate()
	})

	go func() {
		defer log.PanicHandler()

		if deferLoop != nil {
			<-deferLoop
		}

		worker.Backend.Run()
	}()

	worker.PostAction(&types.Configure{Config: acct}, nil)
	worker.PostAction(&types.Connect{}, nil)
	view.SetStatus(state.ConnectionActivity("Connecting..."))
	if acct.CheckMail.Minutes() > 0 {
		view.CheckMailTimer(acct.CheckMail)
	}

	return view, nil
}

func (acct *AccountView) SetStatus(setters ...state.SetStateFunc) {
	for _, fn := range setters {
		fn(&acct.state, acct.SelectedDirectory())
	}
	acct.UpdateStatus()
}

func (acct *AccountView) UpdateStatus() {
	if acct.isSelected() {
		UpdateStatus()
	}
}

func (acct *AccountView) PushStatus(status string, expiry time.Duration) {
	PushStatus(fmt.Sprintf("%s: %s", acct.acct.Name, status), expiry)
}

func (acct *AccountView) PushError(err error) {
	PushError(fmt.Sprintf("%s: %v", acct.acct.Name, err))
}

func (acct *AccountView) PushWarning(warning string) {
	PushWarning(fmt.Sprintf("%s: %s", acct.acct.Name, warning))
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
	if acct.msglist == nil || acct.msglist.Store() == nil {
		return nil, errors.New("init in progress")
	}
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
	return acct == SelectedAccount()
}

func (acct *AccountView) newStore(name string) *lib.MessageStore {
	uiConf := acct.dirlist.UiConfig(name)
	store := lib.NewMessageStore(acct.worker,
		acct.sortCriteria(uiConf),
		uiConf.ThreadingEnabled,
		uiConf.ForceClientThreads,
		uiConf.ClientThreadsDelay,
		uiConf.ReverseOrder,
		uiConf.ReverseThreadOrder,
		uiConf.SortThreadSiblings,
		func(msg *models.MessageInfo) {
			err := hooks.RunHook(&hooks.MailReceived{
				Account: acct.Name(),
				Folder:  name,
				MsgInfo: msg,
			})
			if err != nil {
				msg := fmt.Sprintf("mail-received hook: %s", err)
				PushError(msg)
			}
		}, func() {
			if uiConf.NewMessageBell {
				aerc.Beep()
			}
		},
		acct.updateSplitView,
		acct.dirlist.UiConfig(name).ThreadContext,
	)
	store.SetMarker(marker.New(store))
	return store
}

func (acct *AccountView) onMessage(msg types.WorkerMessage) {
	msg = acct.worker.ProcessMessage(msg)
	switch msg := msg.(type) {
	case *types.Done:
		switch resp := msg.InResponseTo().(type) {
		case *types.Connect, *types.Reconnect:
			acct.SetStatus(state.ConnectionActivity("Listing mailboxes..."))
			log.Infof("[%s] connected.", acct.acct.Name)
			acct.SetStatus(state.SetConnected(true))
			log.Tracef("Listing mailboxes...")
			acct.worker.PostAction(&types.ListDirectories{}, nil)
		case *types.Disconnect:
			acct.dirlist.ClearList()
			acct.msglist.SetStore(nil)
			log.Infof("[%s] disconnected.", acct.acct.Name)
			acct.SetStatus(state.SetConnected(false))
		case *types.OpenDirectory:
			acct.dirlist.Update(msg)
			if store, ok := acct.dirlist.SelectedMsgStore(); ok {
				// If we've opened this dir before, we can re-render it from
				// memory while we wait for the update and the UI feels
				// snappier. If not, we'll unset the store and show the spinner
				// while we download the UID list.
				acct.msglist.SetStore(store)
				acct.Store().Update(msg.InResponseTo())
			} else {
				acct.msglist.SetStore(nil)
			}
		case *types.CreateDirectory:
			store := acct.newStore(resp.Directory)
			acct.dirlist.SetMsgStore(&models.Directory{
				Name: resp.Directory,
			}, store)
			acct.dirlist.Update(msg)
		case *types.RemoveDirectory:
			acct.dirlist.Update(msg)
		case *types.FetchMessageHeaders:
			if acct.newConn {
				acct.checkMailOnStartup()
			}
		case *types.ListDirectories:
			acct.dirlist.Update(msg)
			if dir := acct.dirlist.Selected(); dir != "" {
				acct.dirlist.Select(dir)
				return
			}
			// Nothing selected, select based on config
			dirs := acct.dirlist.List()
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
			acct.newConn = true
		}
	case *types.Directory:
		store, ok := acct.dirlist.MsgStore(msg.Dir.Name)
		if !ok {
			store = acct.newStore(msg.Dir.Name)
		}
		acct.dirlist.SetMsgStore(msg.Dir, store)
	case *types.DirectoryInfo:
		acct.dirlist.Update(msg)
	case *types.DirectoryContents:
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			if acct.msglist.Store() == nil {
				acct.msglist.SetStore(store)
			}
			store.Update(msg)
			acct.SetStatus(state.Threading(store.ThreadedView()))
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
			acct.SetStatus(state.Threading(store.ThreadedView()))
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
		if dir := acct.dirlist.SelectedDirectory(); dir != nil {
			dir.Exists -= len(msg.Uids)
		}
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			store.Update(msg)
		}
	case *types.MessagesCopied:
		acct.updateDirCounts(msg.Destination, msg.Uids)
	case *types.MessagesMoved:
		acct.updateDirCounts(msg.Destination, msg.Uids)
	case *types.LabelList:
		acct.labels = msg.Labels
	case *types.ConnError:
		log.Errorf("[%s] connection error: %v", acct.acct.Name, msg.Error)
		acct.SetStatus(state.SetConnected(false))
		acct.PushError(msg.Error)
		acct.msglist.SetStore(nil)
		acct.worker.PostAction(&types.Reconnect{}, nil)
	case *types.Error:
		log.Errorf("[%s] unexpected error: %v", acct.acct.Name, msg.Error)
		acct.PushError(msg.Error)
	}
	acct.UpdateStatus()
	acct.setTitle()
}

func (acct *AccountView) updateDirCounts(destination string, uids []uint32) {
	// Only update the destination destDir if it is initialized
	if destDir := acct.dirlist.Directory(destination); destDir != nil {
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
			seen := msg.Flags.Has(models.SeenFlag)
			if msg.Flags.Has(models.RecentFlag) {
				recent++
			}
			if !seen {
				unseen++
			}
		}
		if accurate {
			destDir.Recent += recent
			destDir.Unseen += unseen
			destDir.Exists += len(uids)
		} else {
			destDir.Exists += len(uids)
		}
	}
}

func (acct *AccountView) sortCriteria(uiConf *config.UIConfig) []*types.SortCriterion {
	if uiConf == nil {
		return nil
	}
	if len(uiConf.Sort) == 0 {
		return nil
	}
	criteria, err := sort.GetSortCriteria(uiConf.Sort)
	if err != nil {
		acct.PushError(fmt.Errorf("ui sort: %w", err))
		return nil
	}
	return criteria
}

func (acct *AccountView) GetSortCriteria() []*types.SortCriterion {
	return acct.sortCriteria(acct.UiConfig())
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
	log.Debugf("Checking for new mail on account %s", acct.Name())
	acct.SetStatus(state.ConnectionActivity("Checking for new mail..."))
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
			acct.SetStatus(state.ConnectionActivity(""))
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
		defer log.PanicHandler()
		for range acct.ticker.C {
			if !acct.state.Connected {
				continue
			}
			acct.CheckMail()
		}
	}()
}

func (acct *AccountView) closeSplit() {
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

	acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.uiConf))
	acct.grid.AddChild(acct.msglist).At(0, 1)
	ui.Invalidate()
}

func (acct *AccountView) updateSplitView(msg *models.MessageInfo) {
	if acct.splitSize == 0 {
		return
	}
	if acct.splitDebounce != nil {
		acct.splitDebounce.Stop()
	}
	fn := func() {
		if acct.split != nil {
			acct.grid.RemoveChild(acct.split)
			acct.split.Close()
		}
		lib.NewMessageStoreView(msg, false, acct.Store(), CryptoProvider(), DecryptKeys,
			func(view lib.MessageView, err error) {
				if err != nil {
					PushError(err.Error())
					return
				}
				acct.split = NewMessageViewer(acct, view)
				switch acct.splitDir {
				case "split", "hsplit":
					acct.grid.AddChild(acct.split).At(1, 1)
				case "vsplit":
					acct.grid.AddChild(acct.split).At(0, 2)
				}
			})
	}
	acct.splitDebounce = time.AfterFunc(100*time.Millisecond, func() {
		ui.QueueFunc(fn)
	})
}

func (acct *AccountView) SplitSize() int {
	return acct.splitSize
}

func (acct *AccountView) SetSplitSize(n int) {
	if n == 0 {
		acct.closeSplit()
	}
	acct.splitSize = n
}

// Split splits the message list view horizontally. The message list will be n
// rows high. If n is 0, any existing split is removed
func (acct *AccountView) Split(n int) error {
	acct.SetSplitSize(n)
	if acct.splitDir == "split" || n == 0 {
		return nil
	}
	acct.splitDir = "split"
	acct.grid = ui.NewGrid().Rows([]ui.GridSpec{
		// Add 1 so that the splitSize is the number of visible messages
		{Strategy: ui.SIZE_EXACT, Size: func() int { return acct.SplitSize() + 1 }},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return acct.UiConfig().SidebarWidth
		}},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.uiConf)).Span(2, 1)
	acct.grid.AddChild(ui.NewBordered(acct.msglist, ui.BORDER_BOTTOM, acct.uiConf)).At(0, 1)
	acct.split = NewMessageViewer(acct, nil)
	acct.grid.AddChild(acct.split).At(1, 1)
	acct.updateSplitView(acct.msglist.Selected())
	return nil
}

// Vsplit splits the message list view vertically. The message list will be n
// rows wide. If n is 0, any existing split is removed
func (acct *AccountView) Vsplit(n int) error {
	acct.SetSplitSize(n)
	if acct.splitDir == "vsplit" || n == 0 {
		return nil
	}
	acct.splitDir = "vsplit"
	acct.grid = ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return acct.UiConfig().SidebarWidth
		}},
		{Strategy: ui.SIZE_EXACT, Size: acct.SplitSize},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.uiConf)).At(0, 0)
	acct.grid.AddChild(ui.NewBordered(acct.msglist, ui.BORDER_RIGHT, acct.uiConf)).At(0, 1)
	acct.split = NewMessageViewer(acct, nil)
	acct.grid.AddChild(acct.split).At(0, 2)
	acct.updateSplitView(acct.msglist.Selected())
	return nil
}

// setTitle executes the title template and sets the tab title
func (acct *AccountView) setTitle() {
	if acct.tab == nil {
		return
	}

	data := state.NewDataSetter()
	data.SetAccount(acct.acct)
	data.SetFolder(acct.Directories().SelectedDirectory())
	data.SetRUE(acct.dirlist.List(), acct.dirlist.GetRUECount)

	var buf bytes.Buffer
	err := templates.Render(acct.uiConf.TabTitleAccount, &buf, data.Data())
	if err != nil {
		acct.PushError(err)
		return
	}
	acct.tab.SetTitle(buf.String())
}
