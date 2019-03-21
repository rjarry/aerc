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

type vtermKey struct {
	Key  vterm.Key
	Rune rune
	Mod  vterm.Modifier
}

var keyMap map[tcell.Key]vtermKey

func directKey(key vterm.Key) vtermKey {
	return vtermKey{key, 0, vterm.ModNone}
}

func runeMod(r rune, mod vterm.Modifier) vtermKey {
	return vtermKey{vterm.KeyNone, r, mod}
}

func keyMod(key vterm.Key, mod vterm.Modifier) vtermKey {
	return vtermKey{key, 0, mod}
}

func init() {
	keyMap = make(map[tcell.Key]vtermKey)
	keyMap[tcell.KeyCtrlSpace] = runeMod(' ', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlA] = runeMod('a', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlB] = runeMod('b', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlC] = runeMod('c', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlD] = runeMod('d', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlE] = runeMod('e', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlF] = runeMod('f', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlG] = runeMod('g', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlH] = runeMod('h', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlI] = runeMod('i', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlJ] = runeMod('j', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlK] = runeMod('k', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlL] = runeMod('l', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlM] = runeMod('m', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlN] = runeMod('n', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlO] = runeMod('o', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlP] = runeMod('p', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlQ] = runeMod('q', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlR] = runeMod('r', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlS] = runeMod('s', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlT] = runeMod('t', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlU] = runeMod('u', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlV] = runeMod('v', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlW] = runeMod('w', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlX] = runeMod('x', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlY] = runeMod('y', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlZ] = runeMod('z', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlBackslash] = runeMod('\\', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlCarat] = runeMod('^', vterm.ModCtrl)
	keyMap[tcell.KeyCtrlUnderscore] = runeMod('_', vterm.ModCtrl)
	keyMap[tcell.KeyEnter] = directKey(vterm.KeyEnter)
	keyMap[tcell.KeyTab] = directKey(vterm.KeyTab)
	keyMap[tcell.KeyBackspace] = directKey(vterm.KeyBackspace)
	keyMap[tcell.KeyEscape] = directKey(vterm.KeyEscape)
	keyMap[tcell.KeyUp] = directKey(vterm.KeyUp)
	keyMap[tcell.KeyDown] = directKey(vterm.KeyDown)
	keyMap[tcell.KeyLeft] = directKey(vterm.KeyLeft)
	keyMap[tcell.KeyRight] = directKey(vterm.KeyRight)
	keyMap[tcell.KeyInsert] = directKey(vterm.KeyIns)
	keyMap[tcell.KeyDelete] = directKey(vterm.KeyDel)
	keyMap[tcell.KeyHome] = directKey(vterm.KeyHome)
	keyMap[tcell.KeyEnd] = directKey(vterm.KeyEnd)
	keyMap[tcell.KeyPgUp] = directKey(vterm.KeyPageUp)
	keyMap[tcell.KeyPgDn] = directKey(vterm.KeyPageDown)
	for i := 0; i < 64; i++ {
		keyMap[tcell.Key(int(tcell.KeyF1)+i)] =
			directKey(vterm.Key(int(vterm.KeyFunction0) + i))
	}
	keyMap[tcell.KeyTAB] = directKey(vterm.KeyTab)
	keyMap[tcell.KeyESC] = directKey(vterm.KeyEscape)
	keyMap[tcell.KeyDEL] = directKey(vterm.KeyBackspace)
}

type Terminal struct {
	closed       bool
	cmd          *exec.Cmd
	colors       map[tcell.Color]tcell.Color
	ctx          *ui.Context
	cursorPos    vterm.Pos
	cursorShown  bool
	damage       []vterm.Rect
	err          error
	focus        bool
	onInvalidate func(d ui.Drawable)
	pty          *os.File
	start        chan interface{}
	vterm        *vterm.VTerm

	OnClose func(err error)
	OnTitle func(title string)
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
				// These are generally benine errors when the process exits
				term.Close(nil)
				return
			}
			n, err = term.vterm.Write(buf[:n])
			if err != nil {
				term.Close(err)
				return
			}
			term.Invalidate()
		}
	}()
	screen := term.vterm.ObtainScreen()
	screen.OnDamage = term.onDamage
	screen.OnMoveCursor = term.onMoveCursor
	screen.OnSetTermProp = term.onSetTermProp
	screen.Reset(true)

	state := term.vterm.ObtainState()
	term.colors = make(map[tcell.Color]tcell.Color)
	for i := 0; i < 256; i += 1 {
		tcolor := tcell.Color(i)
		var r uint8 = 0
		var g uint8 = 0
		var b uint8 = uint8(i + 1)
		if i < 16 {
			// Set the first 16 colors to predictable near-black RGB values
			state.SetPaletteColor(i,
				vterm.NewVTermColorRGB(gocolor.RGBA{r, g, b, 255}))
		} else {
			// The rest use RGB
			vcolor := state.GetPaletteColor(i)
			r, g, b = vcolor.GetRGB()
		}
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

func (term *Terminal) flushTerminal() {
	buf := make([]byte, 2048)
	for {
		n, err := term.vterm.Read(buf)
		if err != nil {
			term.Close(err)
			return
		}
		if n == 0 {
			break
		}
		n, err = term.pty.Write(buf[:n])
		if err != nil {
			term.Close(err)
			return
		}
	}
}

func (term *Terminal) Close(err error) {
	term.err = err
	if term.vterm != nil {
		term.vterm.Close()
		term.vterm = nil
	}
	if term.pty != nil {
		term.pty.Close()
		term.pty = nil
	}
	if term.cmd != nil && term.cmd.Process != nil {
		term.cmd.Process.Kill()
		term.cmd = nil
	}
	if !term.closed && term.OnClose != nil {
		term.OnClose(err)
	}
	term.closed = true
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
	if term.closed {
		if term.err != nil {
			ui.NewText(term.err.Error()).Strategy(ui.TEXT_CENTER).Draw(ctx)
		} else {
			ui.NewText("Terminal closed").Strategy(ui.TEXT_CENTER).Draw(ctx)
		}
		return
	}

	winsize := pty.Winsize{
		Cols: uint16(ctx.Width()),
		Rows: uint16(ctx.Height()),
	}

	if term.pty == nil {
		term.vterm.SetSize(ctx.Height(), ctx.Width())
		tty, err := pty.StartWithSize(term.cmd, &winsize)
		term.pty = tty
		if err != nil {
			term.Close(err)
			return
		}
		term.start <- nil
	}

	term.ctx = ctx // gross

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

func convertMods(mods tcell.ModMask) vterm.Modifier {
	var (
		ret  uint = 0
		mask uint = uint(mods)
	)
	if mask&uint(tcell.ModShift) > 0 {
		ret |= uint(vterm.ModShift)
	}
	if mask&uint(tcell.ModCtrl) > 0 {
		ret |= uint(vterm.ModCtrl)
	}
	if mask&uint(tcell.ModAlt) > 0 {
		ret |= uint(vterm.ModAlt)
	}
	return vterm.Modifier(ret)
}

func (term *Terminal) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		if event.Key() == tcell.KeyRune {
			term.vterm.KeyboardUnichar(
				event.Rune(), convertMods(event.Modifiers()))
		} else {
			if key, ok := keyMap[event.Key()]; ok {
				if key.Key == vterm.KeyNone {
					term.vterm.KeyboardUnichar(
						key.Rune, key.Mod)
				} else if key.Mod == vterm.ModNone {
					term.vterm.KeyboardKey(key.Key,
						convertMods(event.Modifiers()))
				} else {
					term.vterm.KeyboardKey(key.Key, key.Mod)
				}
			}
		}
		term.flushTerminal()
	}
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

func (term *Terminal) onSetTermProp(prop int, val *vterm.VTermValue) int {
	switch prop {
	case vterm.VTERM_PROP_TITLE:
		if term.OnTitle != nil {
			term.OnTitle(val.String)
		}
	}
	return 1
}
