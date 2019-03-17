package widgets

import (
	gocolor "image/color"
	"os"
	"os/exec"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"

	"git.sr.ht/~sircmpwn/go-libvterm"
	"github.com/gdamore/tcell"
	"github.com/kr/pty"
)

type Terminal struct {
	closed       bool
	cmd          *exec.Cmd
	colors       map[tcell.Color]tcell.Color
	ctx          *ui.Context
	cursorPos    vterm.Pos
	cursorShown  bool
	damage       []vterm.Rect
	focus        bool
	onInvalidate func(d ui.Drawable)
	pty          *os.File
	start        chan interface{}
	vterm        *vterm.VTerm
}

func NewTerminal(cmd *exec.Cmd) (*Terminal, error) {
	term := &Terminal{}
	term.cmd = cmd
	term.vterm = vterm.New(24, 80)
	term.vterm.SetUTF8(true)
	term.start = make(chan interface{})
	go func() {
		<-term.start
		buf := make([]byte, 2048)
		for {
			n, err := term.pty.Read(buf)
			if err != nil {
				term.Close()
			}
			n, err = term.vterm.Write(buf[:n])
			if err != nil {
				term.Close()
			}
			term.Invalidate()
		}
	}()
	screen := term.vterm.ObtainScreen()
	screen.OnDamage = term.onDamage
	screen.OnMoveCursor = term.onMoveCursor
	screen.Reset(true)

	state := term.vterm.ObtainState()
	term.colors = make(map[tcell.Color]tcell.Color)
	for i := 0; i < 16; i += 1 {
		// Set the first 16 colors to predictable near-black RGB values
		tcolor := tcell.Color(i)
		var r uint8 = 0
		var g uint8 = 0
		var b uint8 = uint8(i + 1)
		state.SetPaletteColor(i,
			vterm.NewVTermColorRGB(gocolor.RGBA{r, g, b, 255}))
		term.colors[tcell.NewRGBColor(int32(r), int32(g), int32(b))] = tcolor
	}
	fg, bg := state.GetDefaultColors()
	r, g, b := bg.GetRGB()
	term.colors[tcell.NewRGBColor(
		int32(r), int32(g), int32(b))] = tcell.ColorDefault
	r, g, b = fg.GetRGB()
	term.colors[tcell.NewRGBColor(
		int32(r), int32(g), int32(b))] = tcell.ColorDefault

	return term, nil
}

func (term *Terminal) Close() {
	if term.closed {
		return
	}
	term.closed = true
	term.vterm.Close()
	term.pty.Close()
	term.cmd.Process.Kill()
}

func (term *Terminal) OnInvalidate(cb func(d ui.Drawable)) {
	term.onInvalidate = cb
}

func (term *Terminal) Invalidate() {
	if term.onInvalidate != nil {
		term.onInvalidate(term)
	}
}

func (term *Terminal) Draw(ctx *ui.Context) {
	winsize := pty.Winsize{
		Cols: uint16(ctx.Width()),
		Rows: uint16(ctx.Height()),
	}

	if term.pty == nil {
		term.vterm.SetSize(ctx.Height(), ctx.Width())
		tty, err := pty.StartWithSize(term.cmd, &winsize)
		term.pty = tty
		if err != nil {
			term.Close()
			return
		}
		term.start <- nil
	}

	term.ctx = ctx // gross
	if term.closed {
		return
	}

	rows, cols, err := pty.Getsize(term.pty)
	if err != nil {
		return
	}
	if ctx.Width() != cols || ctx.Height() != rows {
		pty.Setsize(term.pty, &winsize)
		term.vterm.SetSize(ctx.Height(), ctx.Width())
		return
	}

	screen := term.vterm.ObtainScreen()
	screen.Flush()

	type coords struct {
		x int
		y int
	}

	// naive optimization
	visited := make(map[coords]interface{})

	for _, rect := range term.damage {
		for x := rect.StartCol(); x < rect.EndCol() && x < ctx.Width(); x += 1 {

			for y := rect.StartCol(); y < rect.EndCol() && y < ctx.Height(); y += 1 {

				coords := coords{x, y}
				if _, ok := visited[coords]; ok {
					continue
				}
				visited[coords] = nil

				cell, err := screen.GetCellAt(y, x)
				if err != nil {
					continue
				}
				style := term.styleFromCell(cell)
				ctx.Printf(x, y, style, "%s", string(cell.Chars()))
			}
		}
	}
}

func (term *Terminal) Focus(focus bool) {
	term.focus = focus
	term.resetCursor()
}

func (term *Terminal) Event(event tcell.Event) bool {
	// TODO
	return false
}

func (term *Terminal) styleFromCell(cell *vterm.ScreenCell) tcell.Style {
	style := tcell.StyleDefault

	background := cell.Bg()
	r, g, b := background.GetRGB()
	bg := tcell.NewRGBColor(int32(r), int32(g), int32(b))
	foreground := cell.Fg()
	r, g, b = foreground.GetRGB()
	fg := tcell.NewRGBColor(int32(r), int32(g), int32(b))

	if color, ok := term.colors[bg]; ok {
		style = style.Background(color)
	} else {
		style = style.Background(bg)
	}
	if color, ok := term.colors[fg]; ok {
		style = style.Foreground(color)
	} else {
		style = style.Foreground(fg)
	}

	if cell.Attrs().Bold != 0 {
		style = style.Bold(true)
	}
	if cell.Attrs().Underline != 0 {
		style = style.Underline(true)
	}
	if cell.Attrs().Blink != 0 {
		style = style.Blink(true)
	}
	if cell.Attrs().Reverse != 0 {
		style = style.Reverse(true)
	}
	return style
}

func (term *Terminal) onDamage(rect *vterm.Rect) int {
	term.damage = append(term.damage, *rect)
	term.Invalidate()
	return 1
}

func (term *Terminal) resetCursor() {
	if term.ctx != nil && term.focus {
		if !term.cursorShown {
			term.ctx.HideCursor()
		} else {
			term.ctx.SetCursor(term.cursorPos.Col(), term.cursorPos.Row())
		}
	}
}

func (term *Terminal) onMoveCursor(old *vterm.Pos,
	pos *vterm.Pos, visible bool) int {

	term.cursorShown = visible
	term.cursorPos = *pos
	term.resetCursor()
	return 1
}
