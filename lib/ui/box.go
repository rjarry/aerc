package ui

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/mattn/go-runewidth"
)

type Box struct {
	content  Drawable
	title    string
	borders  string
	uiConfig *config.UIConfig
}

func NewBox(
	content Drawable, title, borders string, uiConfig *config.UIConfig,
) *Box {
	if borders == "" || len(borders) < 8 {
		borders = "││┌─┐└─┘"
	}

	b := &Box{
		content:  content,
		title:    title,
		borders:  borders,
		uiConfig: uiConfig,
	}
	return b
}

func (b *Box) Draw(ctx *Context) {
	w := ctx.Width()
	h := ctx.Height()

	style := b.uiConfig.GetStyle(config.STYLE_BORDER)

	box := []rune(b.borders)
	ctx.Fill(0, 0, 1, h, box[0], style)
	ctx.Fill(w-1, 0, 1, h, box[1], style)

	ctx.Printf(0, 0, style, "%c%s%c", box[2], strings.Repeat(string(box[3]), w-2), box[4])
	ctx.Printf(0, h-1, style, "%c%s%c", box[5], strings.Repeat(string(box[6]), w-2), box[7])

	if b.title != "" && w > 4 {
		style = b.uiConfig.GetStyle(config.STYLE_TITLE)
		title := runewidth.Truncate(b.title, w-4, "…")
		ctx.Printf(2, 0, style, "%s", title)
	}

	subctx := ctx.Subcontext(1, 1, w-2, h-2)
	b.content.Draw(subctx)
}

func (b *Box) Invalidate() {
	b.content.Invalidate()
}

func (b *Box) MouseEvent(localX int, localY int, event vaxis.Event) {
	if content, ok := b.content.(Mouseable); ok {
		content.MouseEvent(localX, localY, event)
	}
}

func (b *Box) Event(e vaxis.Event) bool {
	if content, ok := b.content.(Interactive); ok {
		return content.Event(e)
	}
	return false
}

func (b *Box) Focus(_ bool) {
}
