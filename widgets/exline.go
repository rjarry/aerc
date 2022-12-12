package widgets

import (
	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type ExLine struct {
	commit      func(cmd string)
	finish      func()
	tabcomplete func(cmd string) ([]string, string)
	cmdHistory  lib.History
	input       *ui.TextInput
}

func NewExLine(cmd string, commit func(cmd string), finish func(),
	tabcomplete func(cmd string) ([]string, string),
	cmdHistory lib.History,
) *ExLine {
	input := ui.NewTextInput("", config.Ui).Prompt(":").Set(cmd)
	if config.Ui.CompletionPopovers {
		input.TabComplete(
			tabcomplete,
			config.Ui.CompletionDelay,
			config.Ui.CompletionMinChars,
		)
	}
	exline := &ExLine{
		commit:      commit,
		finish:      finish,
		tabcomplete: tabcomplete,
		cmdHistory:  cmdHistory,
		input:       input,
	}
	return exline
}

func (x *ExLine) TabComplete(tabComplete func(string) ([]string, string)) {
	x.input.TabComplete(
		tabComplete,
		config.Ui.CompletionDelay,
		config.Ui.CompletionMinChars,
	)
}

func NewPrompt(prompt string, commit func(text string),
	tabcomplete func(cmd string) ([]string, string),
) *ExLine {
	input := ui.NewTextInput("", config.Ui).Prompt(prompt)
	if config.Ui.CompletionPopovers {
		input.TabComplete(
			tabcomplete,
			config.Ui.CompletionDelay,
			config.Ui.CompletionMinChars,
		)
	}
	exline := &ExLine{
		commit:      commit,
		tabcomplete: tabcomplete,
		cmdHistory:  &nullHistory{input: input},
		input:       input,
	}
	return exline
}

func (ex *ExLine) Invalidate() {
	ui.Invalidate()
}

func (ex *ExLine) Draw(ctx *ui.Context) {
	ex.input.Draw(ctx)
}

func (ex *ExLine) Focus(focus bool) {
	ex.input.Focus(focus)
}

func (ex *ExLine) Event(event tcell.Event) bool {
	if event, ok := event.(*tcell.EventKey); ok {
		switch event.Key() {
		case tcell.KeyEnter, tcell.KeyCtrlJ:
			cmd := ex.input.String()
			ex.input.Focus(false)
			ex.commit(cmd)
			ex.finish()
		case tcell.KeyUp:
			ex.input.Set(ex.cmdHistory.Prev())
			ex.Invalidate()
		case tcell.KeyDown:
			ex.input.Set(ex.cmdHistory.Next())
			ex.Invalidate()
		case tcell.KeyEsc, tcell.KeyCtrlC:
			ex.input.Focus(false)
			ex.cmdHistory.Reset()
			ex.finish()
		default:
			return ex.input.Event(event)
		}
	}
	return true
}

type nullHistory struct {
	input *ui.TextInput
}

func (*nullHistory) Add(string) {}

func (h *nullHistory) Next() string {
	return h.input.String()
}

func (h *nullHistory) Prev() string {
	return h.input.String()
}

func (*nullHistory) Reset() {}
