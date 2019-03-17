package widgets

import (
	"log"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	libui "git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type Aerc struct {
	accounts map[string]*AccountView
	cmd      func(cmd string) error
	grid     *libui.Grid
	tabs     *libui.Tabs
}

func NewAerc(conf *config.AercConfig, logger *log.Logger,
	cmd func(cmd string) error) *Aerc {

	tabs := libui.NewTabs()

	mainGrid := libui.NewGrid().Rows([]libui.GridSpec{
		{libui.SIZE_EXACT, 1},
		{libui.SIZE_WEIGHT, 1},
	}).Columns([]libui.GridSpec{
		{libui.SIZE_EXACT, conf.Ui.SidebarWidth},
		{libui.SIZE_WEIGHT, 1},
	})

	mainGrid.AddChild(libui.NewText("aerc").
		Strategy(libui.TEXT_CENTER).
		Color(tcell.ColorBlack, tcell.ColorWhite))
	mainGrid.AddChild(tabs.TabStrip).At(0, 1)
	mainGrid.AddChild(tabs.TabContent).At(1, 0).Span(1, 2)

	aerc := &Aerc{
		accounts: make(map[string]*AccountView),
		cmd:      cmd,
		grid:     mainGrid,
		tabs:     tabs,
	}

	for _, acct := range conf.Accounts {
		view := NewAccountView(conf, &acct, logger, cmd)
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

func (aerc *Aerc) Event(event tcell.Event) bool {
	acct, _ := aerc.tabs.Tabs[aerc.tabs.Selected].Content.(*AccountView)
	return acct.Event(event)
}

func (aerc *Aerc) SelectedAccount() *AccountView {
	return aerc.accounts[aerc.tabs.Tabs[aerc.tabs.Selected].Name]
}
