package ui

import "git.sr.ht/~rockorager/vaxis"

type Fill struct {
	Rune  rune
	Style vaxis.Style
}

func NewFill(f rune, s vaxis.Style) Fill {
	return Fill{f, s}
}

func (f Fill) Draw(ctx *Context) {
	for x := 0; x < ctx.Width(); x += 1 {
		for y := 0; y < ctx.Height(); y += 1 {
			ctx.SetCell(x, y, f.Rune, f.Style)
		}
	}
}

func (f Fill) Invalidate() {
	// no-op
}
