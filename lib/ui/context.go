package ui

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/gdamore/tcell/v2"
)

// A context allows you to draw in a sub-region of the terminal
type Context struct {
	window    vaxis.Window
	x, y      int
	onPopover func(*Popover)
}

func (ctx *Context) Width() int {
	width, _ := ctx.window.Size()
	return width
}

func (ctx *Context) Height() int {
	_, height := ctx.window.Size()
	return height
}

// returns the vaxis Window for this context
func (ctx *Context) Window() vaxis.Window {
	return ctx.window
}

func NewContext(vx *vaxis.Vaxis, p func(*Popover)) *Context {
	win := vx.Window()
	return &Context{win, 0, 0, p}
}

func (ctx *Context) Subcontext(x, y, width, height int) *Context {
	if x < 0 || y < 0 {
		panic(fmt.Errorf("Attempted to create context with negative offset"))
	}
	win := ctx.window.New(x, y, width, height)
	return &Context{win, x, y, ctx.onPopover}
}

func (ctx *Context) SetCell(x, y int, ch rune, style vaxis.Style) {
	width, height := ctx.window.Size()
	if x >= width || y >= height {
		// no-op when dims are inadequate
		return
	}
	ctx.window.SetCell(x, y, vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: string(ch),
		},
		Style: style,
	})
}

func (ctx *Context) Printf(x, y int, style vaxis.Style,
	format string, a ...interface{},
) int {
	width, height := ctx.window.Size()

	if x >= width || y >= height {
		// no-op when dims are inadequate
		return 0
	}

	str := fmt.Sprintf(format, a...)

	buf := parse.ParseANSI(str)
	buf.ApplyStyle(style)

	old_x := x

	newline := func() bool {
		x = old_x
		y++
		return y < height
	}
	for _, sr := range buf.Runes() {
		switch sr.Value {
		case '\n':
			if !newline() {
				return buf.Len()
			}
		case '\r':
			x = old_x
		default:
			ctx.window.SetCell(x, y, vaxis.Cell{
				Character: vaxis.Character{
					Grapheme: string(sr.Value),
					Width:    sr.Width,
				},
				Style: sr.Style,
			})
			x += sr.Width
			if x == old_x+width {
				if !newline() {
					return buf.Len()
				}
			}
		}
	}

	return buf.Len()
}

func (ctx *Context) Fill(x, y, width, height int, rune rune, style vaxis.Style) {
	win := ctx.window.New(x, y, width, height)
	win.Fill(vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: string(rune),
			Width:    1,
		},
		Style: style,
	})
}

func (ctx *Context) SetCursor(x, y int, style vaxis.CursorStyle) {
	ctx.window.ShowCursor(x, y, style)
}

func (ctx *Context) HideCursor() {
	ctx.window.Vx.HideCursor()
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

// SetContent is used to update the content of the Surface at the given
// location.
func (ctx *Context) SetContent(x int, y int, ch rune, comb []rune, style tcell.Style) {
	g := []rune{ch}
	g = append(g, comb...)
	ctx.window.SetCell(x, y, vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: string(g),
		},
		Style: tcell.VaxisStyle(style),
	})
}

func (ctx *Context) Size() (int, int) {
	return ctx.window.Size()
}
