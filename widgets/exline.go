package widgets

import (
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

// TODO: history
// TODO: tab completion
// TODO: scrolling

type ExLine struct {
	command []rune
	commit  func(cmd string)
	ctx     *ui.Context
	cancel  func()
	cells   int
	index   int
	scroll  int

	onInvalidate func(d ui.Drawable)
}

func NewExLine(commit func(cmd string), cancel func()) *ExLine {
	return &ExLine{
		cancel:  cancel,
		cells:   -1,
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
	ex.ctx = ctx // gross
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	ctx.Printf(0, 0, tcell.StyleDefault, ":%s", string(ex.command))
	cells := runewidth.StringWidth(string(ex.command[:ex.index]))
	if cells != ex.cells {
		ctx.SetCursor(cells+1, 0)
	}
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

func (ex *ExLine) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			ex.backspace()
		case tcell.KeyCtrlD, tcell.KeyDelete:
			ex.deleteChar()
		case tcell.KeyCtrlB, tcell.KeyLeft:
			if ex.index > 0 {
				ex.index--
				ex.Invalidate()
			}
		case tcell.KeyCtrlF, tcell.KeyRight:
			if ex.index < len(ex.command) {
				ex.index++
				ex.Invalidate()
			}
		case tcell.KeyCtrlA, tcell.KeyHome:
			ex.index = 0
			ex.Invalidate()
		case tcell.KeyCtrlE, tcell.KeyEnd:
			ex.index = len(ex.command)
			ex.Invalidate()
		case tcell.KeyCtrlW:
			ex.deleteWord()
		case tcell.KeyEnter:
			if ex.ctx != nil {
				ex.ctx.HideCursor()
			}
			ex.commit(string(ex.command))
		case tcell.KeyEsc, tcell.KeyCtrlC:
			if ex.ctx != nil {
				ex.ctx.HideCursor()
			}
			ex.cancel()
		case tcell.KeyRune:
			ex.insert(event.Rune())
		}
	}
	return true
}
