package widgets

import (
	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type TermHost struct {
	grid *ui.Grid
	term *Terminal
}

// Thin wrapper around terminal which puts it in a grid and passes through
// input events. A bit of a hack tbh
func NewTermHost(term *Terminal, conf *config.AercConfig) *TermHost {
	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_EXACT, conf.Ui.SidebarWidth},
		{ui.SIZE_WEIGHT, 1},
	})
	grid.AddChild(term).At(0, 1)
	return &TermHost{grid, term}
}

func (th *TermHost) Draw(ctx *ui.Context) {
	th.grid.Draw(ctx)
}

func (th TermHost) Invalidate() {
	th.grid.Invalidate()
}

func (th *TermHost) OnInvalidate(fn func(d ui.Drawable)) {
	th.grid.OnInvalidate(func(_ ui.Drawable) {
		fn(th)
	})
}

func (th *TermHost) Event(event tcell.Event) bool {
	return th.term.Event(event)
}

func (th *TermHost) Focus(focus bool) {
	th.term.Focus(focus)
}

func (th *TermHost) Terminal() *Terminal {
	return th.term
}
