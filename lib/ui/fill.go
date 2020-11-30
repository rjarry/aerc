package ui

import (
	"github.com/gdamore/tcell/v2"
)

type Fill rune

func NewFill(f rune) Fill {
	return Fill(f)
}

func (f Fill) Draw(ctx *Context) {
	for x := 0; x < ctx.Width(); x += 1 {
		for y := 0; y < ctx.Height(); y += 1 {
			ctx.SetCell(x, y, rune(f), tcell.StyleDefault)
		}
	}
}

func (f Fill) OnInvalidate(callback func(d Drawable)) {
	// no-op
}

func (f Fill) Invalidate() {
	// no-op
}
