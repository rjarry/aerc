package app

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/hooks"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/marker"
	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/lib/sort"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/vaxis"
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

	split         *MessageViewer
	splitSize     int
	splitDebounce *time.Timer
	splitDir      config.SplitDirection
	splitLoaded   bool

	// Check-mail ticker
	ticker       *time.Ticker
	checkingMail bool
	// Indicates whether the account has a new mail: this is a mail that has
	// arrived since the account tab was last focused (if it's currently
	// focused, the flag is not set).
	hasNew bool
}

func (acct *AccountView) UiConfig() *config.UIConfig {
	if dirlist := acct.Directories(); dirlist != nil {
		return dirlist.UiConfig("")
	}
	return config.Ui.ForAccount(acct.acct.Name)
}

func NewAccountView(
	acct *config.AccountConfig, deferLoop chan struct{},
) (*AccountView, error) {
	view := &AccountView{
		acct: acct,
	}

	worker, err := worker.NewWorker(acct.Source, acct.Name)
	if err != nil {
		SetError(fmt.Sprintf("%s: %s", acct.Name, err))
		log.Errorf("%s: %v", acct.Name, err)
		return view, err
	}
	view.worker = worker

	view.dirlist = NewDirectoryList(acct, worker)

	view.msglist = NewMessageList(view)

	view.Configure()

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

func (acct *AccountView) Configure() {
	acct.dirlist.OnVirtualNode(func() {
		acct.msglist.SetStore(nil)
		acct.Invalidate()
	})
	sidebar := acct.UiConfig().SidebarWidth
	acct.grid = ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return sidebar
		}},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})
	if sidebar > 0 {
		acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.UiConfig()))
	}
	acct.grid.AddChild(acct.msglist).At(0, 1)
	acct.setTitle()

	// handle splits
	if acct.split != nil {
		acct.split.Close()
	}
	splitDirection := acct.splitDir
	acct.splitDir = config.SPLIT_NONE
	switch splitDirection {
	case config.SPLIT_HORIZONTAL:
		acct.Split(acct.SplitSize())
	case config.SPLIT_VERTICAL:
		acct.Vsplit(acct.SplitSize())
	}
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

func (acct *AccountView) Select() {
	for i, widget := range aerc.tabs.TabContent.Children() {
		if widget == acct {
			aerc.SelectTabIndex(i)
		}
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

func (acct *AccountView) MouseEvent(localX int, localY int, event vaxis.Event) {
	acct.grid.MouseEvent(localX, localY, event)
}

func (acct *AccountView) Focus(focus bool) {
	// TODO: Unfocus children I guess
	acct.hasNew = false
	acct.setTitle()
}

func (acct *AccountView) Directories() DirectoryLister {
	return acct.dirlist
}

func (acct *AccountView) SetDirectories(d DirectoryLister) {
	if acct.grid != nil {
		acct.grid.ReplaceChild(acct.dirlist, d)
	}
	acct.dirlist = d
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

func (acct *AccountView) MarkedMessages() ([]models.UID, error) {
	if store := acct.Store(); store != nil {
		return store.Marker().Marked(), nil
	}
	return nil, errors.New("no store available")
}

func (acct *AccountView) SelectedMessagePart() *PartInfo {
	return nil
}

func (acct *AccountView) Terminal() *Terminal {
	if acct.split == nil {
		return nil
	}

	return acct.split.Terminal()
}

func (acct *AccountView) isSelected() bool {
	return acct == SelectedAccount()
}

func (acct *AccountView) newStore(name string) *lib.MessageStore {
	uiConf := acct.dirlist.UiConfig(name)
	dir := acct.dirlist.Directory(name)
	role := ""
	if dir != nil {
		role = string(dir.Role)
	}
	backend := acct.AccountConfig().Backend
	store := lib.NewMessageStore(acct.worker, name,
		func() *config.UIConfig {
			return config.Ui.
				ForAccount(acct.Name()).
				ForFolder(name)
		},
		func(msg *models.MessageInfo) {
			err := hooks.RunHook(&hooks.MailReceived{
				Account: acct.Name(),
				Backend: backend,
				Folder:  name,
				Role:    role,
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
			// Set a new message indicator.
			if aerc.SelectedTab() != acct.tab {
				acct.hasNew = true
			}
		}, func() {
			err := hooks.RunHook(&hooks.MailDeleted{
				Account: acct.Name(),
				Backend: backend,
				Folder:  name,
				Role:    role,
			})
			if err != nil {
				msg := fmt.Sprintf("mail-deleted hook: %s", err)
				PushError(msg)
			}
		}, func(dest string) {
			err := hooks.RunHook(&hooks.MailAdded{
				Account: acct.Name(),
				Backend: backend,
				Folder:  dest,
				Role:    role,
			})
			if err != nil {
				msg := fmt.Sprintf("mail-added hook: %s", err)
				PushError(msg)
			}
		}, func(add []string, remove []string, toggle []string) {
			err := hooks.RunHook(&hooks.TagModified{
				Account: acct.Name(),
				Backend: backend,
				Add:     add,
				Remove:  remove,
				Toggle:  toggle,
			})
			if err != nil {
				msg := fmt.Sprintf("tag-modified hook: %s", err)
				PushError(msg)
			}
		}, func(flagname string) {
			err := hooks.RunHook(&hooks.FlagChanged{
				Account:  acct.Name(),
				Backend:  backend,
				Folder:   acct.SelectedDirectory(),
				Role:     role,
				FlagName: flagname,
			})
			if err != nil {
				msg := fmt.Sprintf("flag-changed hook: %s", err)
				PushError(msg)
			}
		},
		func(msg *models.MessageInfo) {
			acct.updateSplitView(msg)

			auto := false
			if c := acct.AccountConfig(); c != nil {
				r, ok := c.Params["pama-auto-switch"]
				if ok {
					if strings.ToLower(r) == "true" {
						auto = true
					}
				}
			}
			if !auto {
				return
			}
			var name string
			if msg != nil && msg.Envelope != nil {
				name = pama.FromSubject(msg.Envelope.Subject)
			}
			pama.DebouncedSwitchProject(name)
		},
	)
	store.Configure(acct.SortCriteria(uiConf))
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
			log.Tracef("[%s] Listing mailboxes...", acct.acct.Name)
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
			if msg.Unsolicited || msg.ReplaceFlags {
				// This is a server generated message update, e.g. a
				// notification that a message has changed (this will happen
				// mostly for flags according to section 2.3.1.1 alinea 4 in
				// the IMAP4rev1 RFC), or a forced flag update.
				// In case the store and the notification disagree on the Seen
				// flag, trust the notification.
				seen_in_sync := true
				if msg_in_store, ok := store.Messages[msg.Info.Uid]; ok && msg_in_store != nil {
					seen_in_sync = msg_in_store.Flags.Has(models.SeenFlag) ==
						msg.Info.Flags.Has(models.SeenFlag)
				}
				if !seen_in_sync {
					if dir := acct.dirlist.SelectedDirectory(); dir != nil {
						// Our view of Unseen is out-of-sync with the server's;
						// update it now.
						if msg.Info.Flags.Has(models.SeenFlag) {
							dir.Unseen -= 1
						} else {
							dir.Unseen += 1
						}
						dir.Unseen = acct.ensurePositive(dir.Unseen, "Unseen")
					}
				}
			}
			store.Update(msg)
		}
	case *types.MessagesDeleted:
		if dir := acct.dirlist.SelectedDirectory(); dir != nil {
			acct.updateDirCounts(dir.Name, msg.Uids, true)
		}
		if store, ok := acct.dirlist.SelectedMsgStore(); ok {
			store.Update(msg)
		}
	case *types.MessagesCopied:
		acct.updateDirCounts(msg.Destination, msg.Uids, false)
	case *types.MessagesMoved:
		acct.updateDirCounts(msg.Destination, msg.Uids, false)
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

func (acct *AccountView) ensurePositive(val int, name string) int {
	if val < 0 {
		acct.worker.Errorf("Unexpected negative value (%d) for %s", val, name)
		return 0
	}
	return val
}

func (acct *AccountView) updateDirCounts(destination string, uids []models.UID, deleted bool) {
	// Only update the destination destDir if it is initialized
	if destDir := acct.dirlist.Directory(destination); destDir != nil {
		var recent, unseen int
		var accurate bool = true
		store := acct.Store()
		if store == nil {
			// This could happen for example if a disconnection happened,
			// and we can't do anything but bail out.
			accurate = false
		} else {
			for _, uid := range uids {
				// Get the message from the originating store
				msg, ok := store.Messages[uid]
				if !ok {
					continue
				}
				// If message that was not yet loaded is copied
				if msg == nil {
					accurate = false
					break
				}
				if msg.Flags.Has(models.RecentFlag) {
					recent++
				}
				seen := msg.Flags.Has(models.SeenFlag)
				if !seen {
					// If the message is unseen, the directory's current unseen
					// count is off by one and (1) too low if the message is new,
					// or (2) too high if the message has been deleted.
					if !deleted {
						unseen++
					} else {
						unseen--
					}
				}
			}
		}
		if accurate {
			destDir.Recent += recent
			destDir.Unseen += unseen
		}
		if !deleted {
			destDir.Exists += len(uids)
		} else {
			destDir.Exists -= len(uids)
		}
		destDir.Unseen = acct.ensurePositive(destDir.Unseen, "Unseen")
		destDir.Recent = acct.ensurePositive(destDir.Recent, "Recent")
		destDir.Exists = acct.ensurePositive(destDir.Exists, "Exists")
	} else {
		acct.worker.Errorf("Skipping unknown directory %s", destination)
	}
}

func (acct *AccountView) SortCriteria(uiConf *config.UIConfig) []*types.SortCriterion {
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
	return acct.SortCriteria(acct.UiConfig())
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
	acct.splitDir = config.SPLIT_NONE
	acct.split = nil
	acct.grid = ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return acct.UiConfig().SidebarWidth
		}},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.UiConfig()))
	acct.grid.AddChild(acct.msglist).At(0, 1)
	ui.Invalidate()
}

func (acct *AccountView) updateSplitView(msg *models.MessageInfo) {
	uiConf := acct.UiConfig()
	if !acct.splitLoaded {
		switch uiConf.MessageListSplit.Direction {
		case config.SPLIT_HORIZONTAL:
			acct.Split(uiConf.MessageListSplit.Size)
		case config.SPLIT_VERTICAL:
			acct.Vsplit(uiConf.MessageListSplit.Size)
		}
		acct.splitLoaded = true
	}
	if acct.splitSize == 0 || !acct.splitLoaded {
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
				viewer, err := NewMessageViewer(acct, view)
				if err != nil {
					PushError(err.Error())
					return
				}
				acct.split = viewer
				switch acct.splitDir {
				case config.SPLIT_HORIZONTAL:
					acct.grid.AddChild(acct.split).At(1, 1)
				case config.SPLIT_VERTICAL:
					acct.grid.AddChild(acct.split).At(0, 2)
				}
				// If the user wants to, start a timer to mark the message read
				// if it stays in the message viewer longer than the requested
				// delay.
				if !uiConf.AutoMarkReadInSplit {
					return
				}
				acct.splitDebounce = time.AfterFunc(uiConf.AutoMarkReadInSplitDelay, func() {
					if view == nil || view.MessageInfo() == nil || view.Store() == nil {
						return
					}
					view.Store().Flag([]models.UID{view.MessageInfo().Uid}, models.SeenFlag, true, nil)
				})
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

func (acct *AccountView) ToggleHeaders() {
	config.Viewer.ShowHeaders = !config.Viewer.ShowHeaders
	if acct.splitSize == 0 {
		return
	}
	msg, err := acct.SelectedMessage()
	if err != nil {
		log.Debugf("split: load message error: %v", err)
	}
	acct.updateSplitView(msg)
}

// Split splits the message list view horizontally. The message list will be n
// rows high. If n is 0, any existing split is removed
func (acct *AccountView) Split(n int) {
	acct.SetSplitSize(n)
	if acct.splitDir == config.SPLIT_HORIZONTAL || n == 0 {
		return
	}
	acct.splitDir = config.SPLIT_HORIZONTAL
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

	acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.UiConfig())).Span(2, 1)
	acct.grid.AddChild(ui.NewBordered(acct.msglist, ui.BORDER_BOTTOM, acct.UiConfig())).At(0, 1)
	acct.split, _ = NewMessageViewer(acct, nil)
	acct.grid.AddChild(acct.split).At(1, 1)
	msg, err := acct.SelectedMessage()
	if err != nil {
		log.Debugf("split: load message error: %v", err)
	}
	acct.updateSplitView(msg)
}

// Vsplit splits the message list view vertically. The message list will be n
// rows wide. If n is 0, any existing split is removed
func (acct *AccountView) Vsplit(n int) {
	acct.SetSplitSize(n)
	if acct.splitDir == config.SPLIT_VERTICAL || n == 0 {
		return
	}
	acct.splitDir = config.SPLIT_VERTICAL
	acct.grid = ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: func() int {
			return acct.UiConfig().SidebarWidth
		}},
		{Strategy: ui.SIZE_EXACT, Size: acct.SplitSize},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	acct.grid.AddChild(ui.NewBordered(acct.dirlist, ui.BORDER_RIGHT, acct.UiConfig())).At(0, 0)
	acct.grid.AddChild(ui.NewBordered(acct.msglist, ui.BORDER_RIGHT, acct.UiConfig())).At(0, 1)
	acct.split, _ = NewMessageViewer(acct, nil)
	acct.grid.AddChild(acct.split).At(0, 2)
	msg, err := acct.SelectedMessage()
	if err != nil {
		log.Debugf("split: load message error: %v", err)
	}
	acct.updateSplitView(msg)
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
	data.SetState(&acct.state)
	data.SetHasNew(acct.hasNew)

	var buf bytes.Buffer
	err := templates.Render(acct.UiConfig().TabTitleAccount, &buf, data.Data())
	if err != nil {
		acct.PushError(err)
		return
	}
	acct.tab.SetTitle(buf.String())
}
