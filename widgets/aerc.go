package widgets

import (
	"errors"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/google/shlex"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	libui "git.sr.ht/~sircmpwn/aerc/lib/ui"
)

type Aerc struct {
	accounts    map[string]*AccountView
	cmd         func(cmd []string) error
	cmdHistory  lib.History
	complete    func(cmd string) []string
	conf        *config.AercConfig
	focused     libui.Interactive
	grid        *libui.Grid
	logger      *log.Logger
	simulating  int
	statusbar   *libui.Stack
	statusline  *StatusLine
	pendingKeys []config.KeyStroke
	tabs        *libui.Tabs
	beep        func() error
}

func NewAerc(conf *config.AercConfig, logger *log.Logger,
	cmd func(cmd []string) error, complete func(cmd string) []string,
	cmdHistory lib.History) *Aerc {

	tabs := libui.NewTabs()

	statusbar := ui.NewStack()
	statusline := NewStatusLine()
	statusbar.Push(statusline)

	grid := libui.NewGrid().Rows([]libui.GridSpec{
		{libui.SIZE_EXACT, 1},
		{libui.SIZE_WEIGHT, 1},
		{libui.SIZE_EXACT, 1},
	}).Columns([]libui.GridSpec{
		{libui.SIZE_WEIGHT, 1},
	})
	grid.AddChild(tabs.TabStrip)
	grid.AddChild(tabs.TabContent).At(1, 0)
	grid.AddChild(statusbar).At(2, 0)

	aerc := &Aerc{
		accounts:   make(map[string]*AccountView),
		conf:       conf,
		cmd:        cmd,
		cmdHistory: cmdHistory,
		complete:   complete,
		grid:       grid,
		logger:     logger,
		statusbar:  statusbar,
		statusline: statusline,
		tabs:       tabs,
	}

	statusline.SetAerc(aerc)
	conf.Triggers.ExecuteCommand = cmd

	for i, acct := range conf.Accounts {
		view := NewAccountView(conf, &conf.Accounts[i], logger, aerc)
		aerc.accounts[acct.Name] = view
		tabs.Add(view, acct.Name)
	}

	if len(conf.Accounts) == 0 {
		wizard := NewAccountWizard(aerc.Config(), aerc)
		wizard.Focus(true)
		aerc.NewTab(wizard, "New account")
	}

	return aerc
}

func (aerc *Aerc) OnBeep(f func() error) {
	aerc.beep = f
}

func (aerc *Aerc) Beep() {
	if aerc.beep == nil {
		aerc.logger.Printf("should beep, but no beeper")
		return
	}
	if err := aerc.beep(); err != nil {
		aerc.logger.Printf("tried to beep, but could not: %v", err)
	}
}

func (aerc *Aerc) Tick() bool {
	more := false
	for _, acct := range aerc.accounts {
		more = acct.Tick() || more
	}
	return more
}

func (aerc *Aerc) Children() []ui.Drawable {
	return aerc.grid.Children()
}

func (aerc *Aerc) OnInvalidate(onInvalidate func(d libui.Drawable)) {
	aerc.grid.OnInvalidate(func(_ libui.Drawable) {
		onInvalidate(aerc)
	})
}

func (aerc *Aerc) Invalidate() {
	aerc.grid.Invalidate()
}

func (aerc *Aerc) Focus(focus bool) {
	// who cares
}

func (aerc *Aerc) Draw(ctx *libui.Context) {
	aerc.grid.Draw(ctx)
}

func (aerc *Aerc) getBindings() *config.KeyBindings {
	switch view := aerc.SelectedTab().(type) {
	case *AccountView:
		return aerc.conf.Bindings.MessageList
	case *AccountWizard:
		return aerc.conf.Bindings.AccountWizard
	case *Composer:
		switch view.Bindings() {
		case "compose::editor":
			return aerc.conf.Bindings.ComposeEditor
		case "compose::review":
			return aerc.conf.Bindings.ComposeReview
		default:
			return aerc.conf.Bindings.Compose
		}
	case *MessageViewer:
		return aerc.conf.Bindings.MessageView
	case *Terminal:
		return aerc.conf.Bindings.Terminal
	default:
		return aerc.conf.Bindings.Global
	}
}

func (aerc *Aerc) simulate(strokes []config.KeyStroke) {
	aerc.pendingKeys = []config.KeyStroke{}
	aerc.simulating += 1
	for _, stroke := range strokes {
		simulated := tcell.NewEventKey(
			stroke.Key, stroke.Rune, tcell.ModNone)
		aerc.Event(simulated)
	}
	aerc.simulating -= 1
}

func (aerc *Aerc) Event(event tcell.Event) bool {
	if aerc.focused != nil {
		return aerc.focused.Event(event)
	}

	switch event := event.(type) {
	case *tcell.EventKey:
		aerc.statusline.Expire()
		aerc.pendingKeys = append(aerc.pendingKeys, config.KeyStroke{
			Key:  event.Key(),
			Rune: event.Rune(),
		})
		aerc.statusline.Invalidate()
		bindings := aerc.getBindings()
		incomplete := false
		result, strokes := bindings.GetBinding(aerc.pendingKeys)
		switch result {
		case config.BINDING_FOUND:
			aerc.simulate(strokes)
			return true
		case config.BINDING_INCOMPLETE:
			incomplete = true
		case config.BINDING_NOT_FOUND:
		}
		if bindings.Globals {
			result, strokes = aerc.conf.Bindings.Global.
				GetBinding(aerc.pendingKeys)
			switch result {
			case config.BINDING_FOUND:
				aerc.simulate(strokes)
				return true
			case config.BINDING_INCOMPLETE:
				incomplete = true
			case config.BINDING_NOT_FOUND:
			}
		}
		if !incomplete {
			aerc.pendingKeys = []config.KeyStroke{}
			exKey := bindings.ExKey
			if aerc.simulating > 0 {
				// Keybindings still use : even if you change the ex key
				exKey = aerc.conf.Bindings.Global.ExKey
			}
			if event.Key() == exKey.Key && event.Rune() == exKey.Rune {
				aerc.BeginExCommand()
				return true
			}
			interactive, ok := aerc.tabs.Tabs[aerc.tabs.Selected].Content.(ui.Interactive)
			if ok {
				return interactive.Event(event)
			}
			return false
		}
	case *tcell.EventMouse:
		aerc.tabs.MouseEvent(event)
	}
	return false
}

func (aerc *Aerc) Config() *config.AercConfig {
	return aerc.conf
}

func (aerc *Aerc) Logger() *log.Logger {
	return aerc.logger
}

func (aerc *Aerc) SelectedAccount() *AccountView {
	acct, ok := aerc.accounts[aerc.tabs.Tabs[aerc.tabs.Selected].Name]
	if !ok {
		return nil
	}
	return acct
}

func (aerc *Aerc) SelectedTab() ui.Drawable {
	return aerc.tabs.Tabs[aerc.tabs.Selected].Content
}

func (aerc *Aerc) NewTab(drawable ui.Drawable, name string) *ui.Tab {
	tab := aerc.tabs.Add(drawable, name)
	aerc.tabs.Select(len(aerc.tabs.Tabs) - 1)
	return tab
}

func (aerc *Aerc) RemoveTab(tab ui.Drawable) {
	aerc.tabs.Remove(tab)
}

func (aerc *Aerc) ReplaceTab(tabSrc ui.Drawable, tabTarget ui.Drawable, name string) {
	aerc.tabs.Replace(tabSrc, tabTarget, name)
}

func (aerc *Aerc) NextTab() {
	next := aerc.tabs.Selected + 1
	if next >= len(aerc.tabs.Tabs) {
		next = 0
	}
	aerc.tabs.Select(next)
}

func (aerc *Aerc) PrevTab() {
	next := aerc.tabs.Selected - 1
	if next < 0 {
		next = len(aerc.tabs.Tabs) - 1
	}
	aerc.tabs.Select(next)
}

func (aerc *Aerc) SelectTab(name string) bool {
	for i, tab := range aerc.tabs.Tabs {
		if tab.Name == name {
			aerc.tabs.Select(i)
			return true
		}
	}
	return false
}

func (aerc *Aerc) SelectTabIndex(index int) bool {
	for i, _ := range aerc.tabs.Tabs {
		if i == index {
			aerc.tabs.Select(i)
			return true
		}
	}
	return false
}

func (aerc *Aerc) TabNames() []string {
	var names []string
	for _, tab := range aerc.tabs.Tabs {
		names = append(names, tab.Name)
	}
	return names
}

func (aerc *Aerc) SelectPreviousTab() bool {
	return aerc.tabs.SelectPrevious()
}

// TODO: Use per-account status lines, but a global ex line
func (aerc *Aerc) SetStatus(status string) *StatusMessage {
	return aerc.statusline.Set(status)
}

func (aerc *Aerc) PushStatus(text string, expiry time.Duration) *StatusMessage {
	return aerc.statusline.Push(text, expiry)
}

func (aerc *Aerc) PushError(text string) {
	aerc.PushStatus(text, 10*time.Second).Color(tcell.ColorDefault, tcell.ColorRed)
}

func (aerc *Aerc) focus(item libui.Interactive) {
	if aerc.focused == item {
		return
	}
	if aerc.focused != nil {
		aerc.focused.Focus(false)
	}
	aerc.focused = item
	interactive, ok := aerc.tabs.Tabs[aerc.tabs.Selected].Content.(ui.Interactive)
	if item != nil {
		item.Focus(true)
		if ok {
			interactive.Focus(false)
		}
	} else {
		if ok {
			interactive.Focus(true)
		}
	}
}

func (aerc *Aerc) BeginExCommand() {
	previous := aerc.focused
	exline := NewExLine(func(cmd string) {
		parts, err := shlex.Split(cmd)
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
		err = aerc.cmd(parts)
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
		// only add to history if this is an unsimulated command,
		// ie one not executed from a keybinding
		if aerc.simulating == 0 {
			aerc.cmdHistory.Add(cmd)
		}
		aerc.statusbar.Pop()
		aerc.focus(previous)
	}, func() {
		aerc.statusbar.Pop()
		aerc.focus(previous)
	}, func(cmd string) []string {
		return aerc.complete(cmd)
	}, aerc.cmdHistory)
	aerc.statusbar.Push(exline)
	aerc.focus(exline)
}

func (aerc *Aerc) Mailto(addr *url.URL) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	defaults := make(map[string]string)
	defaults["To"] = addr.Opaque
	headerMap := map[string]string{
		"cc":          "Cc",
		"in-reply-to": "In-Reply-To",
		"subject":     "Subject",
	}
	for key, vals := range addr.Query() {
		if header, ok := headerMap[strings.ToLower(key)]; ok {
			defaults[header] = strings.Join(vals, ",")
		}
	}
	composer := NewComposer(aerc.Config(),
		acct.AccountConfig(), acct.Worker(), defaults)
	composer.FocusSubject()
	title := "New email"
	if subj, ok := defaults["Subject"]; ok {
		title = subj
		composer.FocusTerminal()
	}
	tab := aerc.NewTab(composer, title)
	composer.OnHeaderChange("Subject", func(subject string) {
		if subject == "" {
			tab.Name = "New email"
		} else {
			tab.Name = subject
		}
		tab.Content.Invalidate()
	})
	return nil
}
