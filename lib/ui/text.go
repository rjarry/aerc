package ui

import (
	"git.sr.ht/~rockorager/vaxis"
	"github.com/mattn/go-runewidth"
)

const (
	TEXT_LEFT   = iota
	TEXT_CENTER = iota
	TEXT_RIGHT  = iota
)

type Text struct {
	text     string
	strategy uint
	style    vaxis.Style
}

func NewText(text string, style vaxis.Style) *Text {
	return &Text{
		text:  text,
		style: style,
	}
}

func (t *Text) Text(text string) *Text {
	t.text = text
	t.Invalidate()
	return t
}

func (t *Text) Strategy(strategy uint) *Text {
	t.strategy = strategy
	t.Invalidate()
	return t
}

func (t *Text) Draw(ctx *Context) {
	size := runewidth.StringWidth(t.text)
	x := 0
	if t.strategy == TEXT_CENTER {
		x = (ctx.Width() - size) / 2
	}
	if t.strategy == TEXT_RIGHT {
		x = ctx.Width() - size
	}
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', t.style)
	ctx.Printf(x, 0, t.style, "%s", t.text)
}

func (t *Text) Invalidate() {
	Invalidate()
}
