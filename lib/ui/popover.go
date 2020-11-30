package ui

import "github.com/gdamore/tcell/v2"

type Popover struct {
	x, y, width, height int
	content             Drawable
}

func (p *Popover) Draw(ctx *Context) {
	var subcontext *Context

	// trim desired width to fit
	width := p.width
	if p.x+p.width > ctx.Width() {
		width = ctx.Width() - p.x
	}

	if p.y+p.height+1 < ctx.Height() {
		// draw below
		subcontext = ctx.Subcontext(p.x, p.y+1, width, p.height)
	} else if p.y-p.height >= 0 {
		// draw above
		subcontext = ctx.Subcontext(p.x, p.y-p.height, width, p.height)
	} else {
		// can't fit entirely above or below, so find the largest available
		// vertical space and shrink to fit
		if p.y > ctx.Height()-p.y {
			// there is more space above than below
			height := p.y
			subcontext = ctx.Subcontext(p.x, 0, width, height)
		} else {
			// there is more space below than above
			height := ctx.Height() - p.y
			subcontext = ctx.Subcontext(p.x, p.y+1, width, height-1)
		}
	}
	p.content.Draw(subcontext)
}

func (p *Popover) Event(e tcell.Event) bool {
	if di, ok := p.content.(DrawableInteractive); ok {
		return di.Event(e)
	}
	return false
}

func (p *Popover) Focus(f bool) {
	if di, ok := p.content.(DrawableInteractive); ok {
		di.Focus(f)
	}
}

func (p *Popover) Invalidate() {
	p.content.Invalidate()
}

func (p *Popover) OnInvalidate(f func(Drawable)) {
	p.content.OnInvalidate(func(_ Drawable) {
		f(p)
	})
}
