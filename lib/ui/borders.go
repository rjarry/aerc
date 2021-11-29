package ui

import (
	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
)

const (
	BORDER_LEFT   = 1 << iota
	BORDER_TOP    = 1 << iota
	BORDER_RIGHT  = 1 << iota
	BORDER_BOTTOM = 1 << iota
)

type Bordered struct {
	Invalidatable
	borders      uint
	content      Drawable
	onInvalidate func(d Drawable)
	uiConfig     config.UIConfig
}

func NewBordered(
	content Drawable, borders uint, uiConfig config.UIConfig) *Bordered {
	b := &Bordered{
		borders:  borders,
		content:  content,
		uiConfig: uiConfig,
	}
	content.OnInvalidate(b.contentInvalidated)
	return b
}

func (bordered *Bordered) contentInvalidated(d Drawable) {
	bordered.Invalidate()
}

func (bordered *Bordered) Children() []Drawable {
	return []Drawable{bordered.content}
}

func (bordered *Bordered) Invalidate() {
	bordered.DoInvalidate(bordered)
}

func (bordered *Bordered) Draw(ctx *Context) {
	x := 0
	y := 0
	width := ctx.Width()
	height := ctx.Height()
	style := bordered.uiConfig.GetStyle(config.STYLE_BORDER)
	verticalChar := bordered.uiConfig.BorderCharVertical
	horizontalChar := bordered.uiConfig.BorderCharHorizontal

	if bordered.borders&BORDER_LEFT != 0 {
		ctx.Fill(0, 0, 1, ctx.Height(), verticalChar, style)
		x += 1
		width -= 1
	}
	if bordered.borders&BORDER_TOP != 0 {
		ctx.Fill(0, 0, ctx.Width(), 1, horizontalChar, style)
		y += 1
		height -= 1
	}
	if bordered.borders&BORDER_RIGHT != 0 {
		ctx.Fill(ctx.Width()-1, 0, 1, ctx.Height(), verticalChar, style)
		width -= 1
	}
	if bordered.borders&BORDER_BOTTOM != 0 {
		ctx.Fill(0, ctx.Height()-1, ctx.Width(), 1, horizontalChar, style)
		height -= 1
	}
	subctx := ctx.Subcontext(x, y, width, height)
	bordered.content.Draw(subctx)
}

func (bordered *Bordered) MouseEvent(localX int, localY int, event tcell.Event) {
	switch content := bordered.content.(type) {
	case Mouseable:
		content.MouseEvent(localX, localY, event)
	}
}
