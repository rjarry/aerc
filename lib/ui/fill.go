package ui

import (
	"github.com/gdamore/tcell/v2"
)

type Fill struct {
	Rune  rune
	Style tcell.Style
}

func NewFill(f rune, s tcell.Style) Fill {
	return Fill{f, s}
}

func (f Fill) Draw(ctx *Context) {
	for x := 0; x < ctx.Width(); x += 1 {
		for y := 0; y < ctx.Height(); y += 1 {
			ctx.SetCell(x, y, f.Rune, f.Style)
		}
	}
}

func (f Fill) OnInvalidate(callback func(d Drawable)) {
	// no-op
}

func (f Fill) Invalidate() {
	// no-op
}
