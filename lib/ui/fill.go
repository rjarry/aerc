package ui

import (
	tb "github.com/nsf/termbox-go"
)

type Fill rune

func NewFill(f rune) Fill {
	return Fill(f)
}

func (f Fill) Draw(ctx *Context) {
	for x := 0; x < ctx.Width(); x += 1 {
		for y := 0; y < ctx.Height(); y += 1 {
			ctx.SetCell(x, y, rune(f), tb.ColorDefault, tb.ColorDefault)
		}
	}
}

func (f Fill) OnInvalidate(callback func(d Drawable)) {
	// no-op
}

func (f Fill) Invalidate() {
	// no-op
}
