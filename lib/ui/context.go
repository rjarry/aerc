package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
	"github.com/mattn/go-runewidth"
)

// A context allows you to draw in a sub-region of the terminal
type Context struct {
	screen    tcell.Screen
	viewport  *views.ViewPort
	x, y      int
	onPopover func(*Popover)
}

func (ctx *Context) X() int {
	x, _, _, _ := ctx.viewport.GetPhysical()
	return x
}

func (ctx *Context) Y() int {
	_, y, _, _ := ctx.viewport.GetPhysical()
	return y
}

func (ctx *Context) Width() int {
	width, _ := ctx.viewport.Size()
	return width
}

func (ctx *Context) Height() int {
	_, height := ctx.viewport.Size()
	return height
}

func NewContext(width, height int, screen tcell.Screen, p func(*Popover)) *Context {
	vp := views.NewViewPort(screen, 0, 0, width, height)
	return &Context{screen, vp, 0, 0, p}
}

func (ctx *Context) Subcontext(x, y, width, height int) *Context {
	vp_width, vp_height := ctx.viewport.Size()
	if x < 0 || y < 0 {
		panic(fmt.Errorf("Attempted to create context with negative offset"))
	}
	if x+width > vp_width || y+height > vp_height {
		panic(fmt.Errorf("Attempted to create context larger than parent"))
	}
	vp := views.NewViewPort(ctx.viewport, x, y, width, height)
	return &Context{ctx.screen, vp, ctx.x + x, ctx.y + y, ctx.onPopover}
}

func (ctx *Context) SetCell(x, y int, ch rune, style tcell.Style) {
	width, height := ctx.viewport.Size()
	if x >= width || y >= height {
		panic(fmt.Errorf("Attempted to draw outside of context"))
	}
	crunes := []rune{}
	ctx.viewport.SetContent(x, y, ch, crunes, style)
}

func (ctx *Context) Printf(x, y int, style tcell.Style,
	format string, a ...interface{}) int {
	width, height := ctx.viewport.Size()

	if x >= width || y >= height {
		panic(fmt.Errorf("Attempted to draw outside of context"))
	}

	str := fmt.Sprintf(format, a...)

	old_x := x

	newline := func() bool {
		x = old_x
		y++
		return y < height
	}
	for _, ch := range str {
		switch ch {
		case '\n':
			if !newline() {
				return runewidth.StringWidth(str)
			}
		case '\r':
			x = old_x
		default:
			crunes := []rune{}
			ctx.viewport.SetContent(x, y, ch, crunes, style)
			x += runewidth.RuneWidth(ch)
			if x == old_x+width {
				if !newline() {
					return runewidth.StringWidth(str)
				}
			}
		}
	}

	return runewidth.StringWidth(str)
}

func (ctx *Context) Fill(x, y, width, height int, rune rune, style tcell.Style) {
	vp := views.NewViewPort(ctx.viewport, x, y, width, height)
	vp.Fill(rune, style)
}

func (ctx *Context) SetCursor(x, y int) {
	ctx.screen.ShowCursor(ctx.x+x, ctx.y+y)
}

func (ctx *Context) HideCursor() {
	ctx.screen.HideCursor()
}

func (ctx *Context) Popover(x, y, width, height int, d Drawable) {
	ctx.onPopover(&Popover{
		x:       ctx.x + x,
		y:       ctx.y + y,
		width:   width,
		height:  height,
		content: d,
	})
}
