package ui

import (
	"fmt"

	"github.com/nsf/termbox-go"
)

// A context allows you to draw in a sub-region of the terminal
type Context struct {
	x      int
	y      int
	width  int
	height int
}

func (ctx *Context) Width() int {
	return ctx.width
}

func (ctx *Context) Height() int {
	return ctx.height
}

func NewContext(width, height int) *Context {
	return &Context{0, 0, width, height}
}

func (ctx *Context) Subcontext(x, y, width, height int) *Context {
	if x+width > ctx.width || y+height > ctx.height {
		panic(fmt.Errorf("Attempted to create context larger than parent"))
	}
	return &Context{
		x:      ctx.x + x,
		y:      ctx.y + y,
		width:  width,
		height: height,
	}
}

func (ctx *Context) SetCell(x, y int, ch rune, fg, bg termbox.Attribute) {
	if x >= ctx.width || y >= ctx.height {
		panic(fmt.Errorf("Attempted to draw outside of context"))
	}
	termbox.SetCell(ctx.x+x, ctx.y+y, ch, fg, bg)
}

func (ctx *Context) Printf(x, y int, ref termbox.Cell,
	format string, a ...interface{}) {

	if x >= ctx.width || y >= ctx.height {
		panic(fmt.Errorf("Attempted to draw outside of context"))
	}

	str := fmt.Sprintf(format, a...)

	x += ctx.x
	y += ctx.y
	old_x := x

	newline := func() bool {
		x = old_x
		y++
		return y < ctx.height
	}
	for _, ch := range str {
		switch ch {
		case '\n':
			if !newline() {
				return
			}
		case '\r':
			x = old_x
		default:
			termbox.SetCell(x, y, ch, ref.Fg, ref.Bg)
			x++
			if x == old_x+ctx.width {
				if !newline() {
					return
				}
			}
		}
	}
}

func (ctx *Context) Fill(x, y, width, height int, ref termbox.Cell) {
	_x := x
	for ; y < height && y < ctx.height; y++ {
		for ; x < width && x < ctx.width; x++ {
			ctx.SetCell(x, y, ref.Ch, ref.Fg, ref.Bg)
		}
		x = _x
	}
}
