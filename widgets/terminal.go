package widgets

import (
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
	ctx          *ui.Context
	cursorPos    vterm.Pos
	cursorShown  bool
	damage       []vterm.Rect
	focus        bool
	onInvalidate func(d ui.Drawable)
	pty          *os.File
	vterm        *vterm.VTerm
}

func NewTerminal(cmd *exec.Cmd) (*Terminal, error) {
	term := &Terminal{}
	term.cmd = cmd
	tty, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	term.pty = tty
	rows, cols, err := pty.Getsize(term.pty)
	if err != nil {
		return nil, err
	}
	term.vterm = vterm.New(rows, cols)
	term.vterm.SetUTF8(true)
	go func() {
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
	term.ctx = ctx // gross
	if term.closed {
		return
	}

	rows, cols, err := pty.Getsize(term.pty)
	if err != nil {
		return
	}
	if ctx.Width() != cols || ctx.Height() != rows {
		winsize := pty.Winsize{
			Cols: uint16(ctx.Width()),
			Rows: uint16(ctx.Height()),
		}
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
				style := styleFromCell(cell)
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

func styleFromCell(cell *vterm.ScreenCell) tcell.Style {
	background := cell.Bg()
	br, bg, bb := background.GetRGB()
	foreground := cell.Fg()
	fr, fg, fb := foreground.GetRGB()
	style := tcell.StyleDefault.
		Background(tcell.NewRGBColor(int32(br), int32(bg), int32(bb))).
		Foreground(tcell.NewRGBColor(int32(fr), int32(fg), int32(fb)))
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
