package widgets

import (
	"fmt"
	"log"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	libui "git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type Aerc struct {
	accounts map[string]*AccountView
	grid     *libui.Grid
	tabs     *libui.Tabs
}

func NewAerc(conf *config.AercConfig, logger *log.Logger) *Aerc {
	tabs := libui.NewTabs()

	mainGrid := libui.NewGrid().Rows([]libui.GridSpec{
		{libui.SIZE_EXACT, 1},
		{libui.SIZE_WEIGHT, 1},
	}).Columns([]libui.GridSpec{
		{libui.SIZE_EXACT, 20},
		{libui.SIZE_WEIGHT, 1},
	})

	// TODO: Grab sidebar size from config and via :set command
	mainGrid.AddChild(libui.NewText("aerc").
		Strategy(libui.TEXT_CENTER).
		Color(tcell.ColorBlack, tcell.ColorWhite))
	mainGrid.AddChild(tabs.TabStrip).At(0, 1)
	mainGrid.AddChild(tabs.TabContent).At(1, 0).Span(1, 2)

	aerc := &Aerc{
		accounts: make(map[string]*AccountView),
		grid:     mainGrid,
		tabs:     tabs,
	}

	for _, acct := range conf.Accounts {
		view := NewAccountView(&acct, logger, aerc.RunCommand)
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

func (aerc *Aerc) Draw(ctx *libui.Context) {
	aerc.grid.Draw(ctx)
}

func (aerc *Aerc) Event(event tcell.Event) bool {
	acct, _ := aerc.tabs.Tabs[aerc.tabs.Selected].Content.(*AccountView)
	return acct.Event(event)
}

func (aerc *Aerc) RunCommand(cmd string) error {
	// TODO
	return fmt.Errorf("TODO: execute '%s'", cmd)
}
