package widgets

import (
	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type ExLine struct {
	ui.Invalidatable
	cancel func()
	commit func(cmd string)
	ctx    *ui.Context
	input  *ui.TextInput
}

func NewExLine(commit func(cmd string), cancel func()) *ExLine {
	input := ui.NewTextInput().Prompt(":")
	exline := &ExLine{
		cancel: cancel,
		commit: commit,
		input:  input,
	}
	input.OnInvalidate(func(d ui.Drawable) {
		exline.Invalidate()
	})
	return exline
}

func (ex *ExLine) Invalidate() {
	ex.DoInvalidate(ex)
}

func (ex *ExLine) Draw(ctx *ui.Context) {
	ex.ctx = ctx // gross
	ex.input.Draw(ctx)
}

func (ex *ExLine) Focus(focus bool) {
	ex.input.Focus(focus)
}

func (ex *ExLine) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyEnter:
			if ex.ctx != nil {
				ex.ctx.HideCursor()
			}
			ex.commit(ex.input.String())
		case tcell.KeyEsc, tcell.KeyCtrlC:
			if ex.ctx != nil {
				ex.ctx.HideCursor()
			}
			ex.cancel()
		default:
			return ex.input.Event(event)
		}
	}
	return true
}
