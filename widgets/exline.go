package widgets

import (
	"github.com/mattn/go-runewidth"
	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

// TODO: history
// TODO: tab completion
// TODO: scrolling

type ExLine struct {
	command []rune
	commit  func(cmd string)
	cancel  func()
	index   int
	scroll  int

	onInvalidate func(d ui.Drawable)
}

func NewExLine(commit func (cmd string), cancel func()) *ExLine {
	return &ExLine{
		cancel:  cancel,
		commit:  commit,
		command: []rune{},
	}
}

func (ex *ExLine) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	ex.onInvalidate = onInvalidate
}

func (ex *ExLine) Invalidate() {
	if ex.onInvalidate != nil {
		ex.onInvalidate(ex)
	}
}

func (ex *ExLine) Draw(ctx *ui.Context) {
	cell := tb.Cell{
		Fg: tb.ColorDefault,
		Bg: tb.ColorDefault,
		Ch: ' ',
	}
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), cell)
	ctx.Printf(0, 0, cell, ":%s", string(ex.command))
	cells := runewidth.StringWidth(string(ex.command[:ex.index]))
	tb.SetCursor(ctx.X()+cells+1, ctx.Y())
}

func (ex *ExLine) insert(ch rune) {
	left := ex.command[:ex.index]
	right := ex.command[ex.index:]
	ex.command = append(left, append([]rune{ch}, right...)...)
	ex.index++
	ex.Invalidate()
}

func (ex *ExLine) deleteWord() {
	// TODO: Break on any of / " '
	if len(ex.command) == 0 {
		return
	}
	i := ex.index - 1
	if ex.command[i] == ' ' {
		i--
	}
	for ; i >= 0; i-- {
		if ex.command[i] == ' ' {
			break
		}
	}
	ex.command = append(ex.command[:i+1], ex.command[ex.index:]...)
	ex.index = i + 1
	ex.Invalidate()
}

func (ex *ExLine) deleteChar() {
	if len(ex.command) > 0 && ex.index != len(ex.command) {
		ex.command = append(ex.command[:ex.index], ex.command[ex.index+1:]...)
		ex.Invalidate()
	}
}

func (ex *ExLine) backspace() {
	if len(ex.command) > 0 && ex.index != 0 {
		ex.command = append(ex.command[:ex.index-1], ex.command[ex.index:]...)
		ex.index--
		ex.Invalidate()
	}
}

func (ex *ExLine) Event(event tb.Event) bool {
	switch event.Type {
	case tb.EventKey:
		switch event.Key {
		case tb.KeySpace:
			ex.insert(' ')
		case tb.KeyBackspace, tb.KeyBackspace2:
			ex.backspace()
		case tb.KeyCtrlD, tb.KeyDelete:
			ex.deleteChar()
		case tb.KeyCtrlB, tb.KeyArrowLeft:
			if ex.index > 0 {
				ex.index--
				ex.Invalidate()
			}
		case tb.KeyCtrlF, tb.KeyArrowRight:
			if ex.index < len(ex.command) {
				ex.index++
				ex.Invalidate()
			}
		case tb.KeyCtrlA, tb.KeyHome:
			ex.index = 0
			ex.Invalidate()
		case tb.KeyCtrlE, tb.KeyEnd:
			ex.index = len(ex.command)
			ex.Invalidate()
		case tb.KeyCtrlW:
			ex.deleteWord()
		case tb.KeyEnter:
			tb.HideCursor()
			ex.commit(string(ex.command))
		case tb.KeyEsc, tb.KeyCtrlC:
			tb.HideCursor()
			ex.cancel()
		default:
			if event.Ch != 0 {
				ex.insert(event.Ch)
			}
		}
	}
	return true
}
