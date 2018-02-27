package ui

import (
	"github.com/mattn/go-runewidth"
	tb "github.com/nsf/termbox-go"
)

const (
	TEXT_LEFT   = iota
	TEXT_CENTER = iota
	TEXT_RIGHT  = iota
)

type Text struct {
	text         string
	strategy     uint
	fg           tb.Attribute
	bg           tb.Attribute
	onInvalidate func(d Drawable)
}

func NewText(text string) *Text {
	return &Text{text: text}
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

func (t *Text) Color(fg tb.Attribute, bg tb.Attribute) *Text {
	t.fg = fg
	t.bg = bg
	t.Invalidate()
	return t
}

func (t *Text) Draw(ctx *Context) {
	size := runewidth.StringWidth(t.text)
	cell := tb.Cell{
		Ch: ' ',
		Fg: t.fg,
		Bg: t.bg,
	}
	x := 0
	if t.strategy == TEXT_CENTER {
		x = (ctx.Width() - size) / 2
	}
	if t.strategy == TEXT_RIGHT {
		x = ctx.Width() - size
	}
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), cell)
	ctx.Printf(x, 0, cell, "%s", t.text)
}

func (t *Text) OnInvalidate(onInvalidate func(d Drawable)) {
	t.onInvalidate = onInvalidate
}

func (t *Text) Invalidate() {
	if t.onInvalidate != nil {
		t.onInvalidate(t)
	}
}
