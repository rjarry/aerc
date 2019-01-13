package widgets

import (
	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type AccountView struct {
	conf         *config.AccountConfig
	grid         *ui.Grid
	onInvalidate func(d ui.Drawable)
}

func NewAccountView(conf *config.AccountConfig,
	statusbar ui.Drawable) *AccountView {

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_EXACT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_EXACT, 20},
		{ui.SIZE_WEIGHT, 1},
	})
	grid.AddChild(ui.NewBordered(
		ui.NewFill('s'), ui.BORDER_RIGHT)).Span(2, 1)
	grid.AddChild(ui.NewFill('.')).At(0, 1)
	grid.AddChild(statusbar).At(1, 1)
	return &AccountView{conf: conf, grid: grid}
}

func (acct *AccountView) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	acct.grid.OnInvalidate(onInvalidate)
}

func (acct *AccountView) Invalidate() {
	acct.grid.Invalidate()
}

func (acct *AccountView) Draw(ctx *ui.Context) {
	acct.grid.Draw(ctx)
}
