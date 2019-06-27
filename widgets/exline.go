package widgets

import (
	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/lib/ui"
)

type ExLine struct {
	ui.Invalidatable
	cancel      func()
	commit      func(cmd string)
	tabcomplete func(cmd string) []string
	input       *ui.TextInput
}

func NewExLine(commit func(cmd string), cancel func(),
	tabcomplete func(cmd string) []string) *ExLine {

	input := ui.NewTextInput("").Prompt(":")
	exline := &ExLine{
		cancel:      cancel,
		commit:      commit,
		tabcomplete: tabcomplete,
		input:       input,
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
			ex.input.Focus(false)
			ex.commit(ex.input.String())
		case tcell.KeyEsc, tcell.KeyCtrlC:
			ex.input.Focus(false)
			ex.cancel()
		case tcell.KeyTab:
			complete := ex.tabcomplete(ex.input.StringLeft())
			if len(complete) == 1 {
				ex.input.Set(complete[0] + " " + ex.input.StringRight())
			}
			ex.Invalidate()
		default:
			return ex.input.Event(event)
		}
	}
	return true
}
