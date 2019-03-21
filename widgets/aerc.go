package widgets

import (
	"log"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	libui "git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type Aerc struct {
	accounts    map[string]*AccountView
	cmd         func(cmd string) error
	conf        *config.AercConfig
	focused     libui.Interactive
	grid        *libui.Grid
	logger      *log.Logger
	statusbar   *libui.Stack
	statusline  *StatusLine
	pendingKeys []config.KeyStroke
	tabs        *libui.Tabs
}

func NewAerc(conf *config.AercConfig, logger *log.Logger,
	cmd func(cmd string) error) *Aerc {

	tabs := libui.NewTabs()

	statusbar := ui.NewStack()
	statusline := NewStatusLine()
	statusbar.Push(statusline)

	grid := libui.NewGrid().Rows([]libui.GridSpec{
		{libui.SIZE_EXACT, 1},
		{libui.SIZE_WEIGHT, 1},
		{libui.SIZE_EXACT, 1},
	}).Columns([]libui.GridSpec{
		{libui.SIZE_EXACT, conf.Ui.SidebarWidth},
		{libui.SIZE_WEIGHT, 1},
	})
	grid.AddChild(statusbar).At(2, 1)
	// Minor hack
	grid.AddChild(libui.NewBordered(
		libui.NewFill(' '), libui.BORDER_RIGHT)).At(2, 0)

	grid.AddChild(libui.NewText("aerc").
		Strategy(libui.TEXT_CENTER).
		Color(tcell.ColorBlack, tcell.ColorWhite))
	grid.AddChild(tabs.TabStrip).At(0, 1)
	grid.AddChild(tabs.TabContent).At(1, 0).Span(1, 2)

	aerc := &Aerc{
		accounts:   make(map[string]*AccountView),
		conf:       conf,
		cmd:        cmd,
		grid:       grid,
		logger:     logger,
		statusbar:  statusbar,
		statusline: statusline,
		tabs:       tabs,
	}

	for _, acct := range conf.Accounts {
		view := NewAccountView(conf, &acct, logger, aerc)
		aerc.accounts[acct.Name] = view
		tabs.Add(view, acct.Name)
	}

	return aerc
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
	switch aerc.SelectedTab().(type) {
	case *AccountView:
		return aerc.conf.Bindings.MessageList
	case *TermHost:
		return aerc.conf.Bindings.Terminal
	default:
		return aerc.conf.Bindings.Global
	}
}

func (aerc *Aerc) simulate(strokes []config.KeyStroke) {
	aerc.pendingKeys = []config.KeyStroke{}
	for _, stroke := range strokes {
		simulated := tcell.NewEventKey(
			stroke.Key, stroke.Rune, tcell.ModNone)
		aerc.Event(simulated)
	}
}

func (aerc *Aerc) Event(event tcell.Event) bool {
	if aerc.focused != nil {
		return aerc.focused.Event(event)
	}

	switch event := event.(type) {
	case *tcell.EventKey:
		aerc.pendingKeys = append(aerc.pendingKeys, config.KeyStroke{
			Key:  event.Key(),
			Rune: event.Rune(),
		})
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
			if event.Key() == bindings.ExKey.Key &&
				event.Rune() == bindings.ExKey.Rune {

				aerc.BeginExCommand()
				return true
			}
			interactive, ok := aerc.tabs.Tabs[aerc.tabs.Selected].Content.(ui.Interactive)
			if ok {
				return interactive.Event(event)
			}
			return false
		}
	}
	return false
}

func (aerc *Aerc) Config() *config.AercConfig {
	return aerc.conf
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

// TODO: Use per-account status lines, but a global ex line
func (aerc *Aerc) SetStatus(status string) *StatusMessage {
	return aerc.statusline.Set(status)
}

func (aerc *Aerc) PushStatus(text string, expiry time.Duration) *StatusMessage {
	return aerc.statusline.Push(text, expiry)
}

func (aerc *Aerc) focus(item libui.Interactive) {
	if aerc.focused == item {
		return
	}
	if aerc.focused != nil {
		aerc.focused.Focus(false)
	}
	aerc.focused = item
	if item != nil {
		item.Focus(true)
	}
}

func (aerc *Aerc) BeginExCommand() {
	previous := aerc.focused
	exline := NewExLine(func(cmd string) {
		err := aerc.cmd(cmd)
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorRed, tcell.ColorWhite)
		}
		aerc.statusbar.Pop()
		aerc.focus(previous)
	}, func() {
		aerc.statusbar.Pop()
		aerc.focus(previous)
	})
	aerc.statusbar.Push(exline)
	aerc.focus(exline)
}
