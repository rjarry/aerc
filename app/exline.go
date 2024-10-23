package app

import (
	"context"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/go-opt/v2"
	"git.sr.ht/~rockorager/vaxis"
)

type ExLine struct {
	commit      func(cmd string)
	finish      func()
	tabcomplete func(ctx context.Context, cmd string) ([]opt.Completion, string)
	cmdHistory  lib.History
	input       *ui.TextInput
}

func NewExLine(cmd string, commit func(cmd string), finish func(),
	tabcomplete func(ctx context.Context, cmd string) ([]opt.Completion, string),
	cmdHistory lib.History,
) *ExLine {
	input := ui.NewTextInput("", config.Ui).Prompt(":").Set(cmd)
	if config.Ui.CompletionPopovers {
		input.TabComplete(
			tabcomplete,
			config.Ui.CompletionDelay,
			config.Ui.CompletionMinChars,
			&config.Binds.Global.CompleteKey,
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

func (x *ExLine) TabComplete(tabComplete func(context.Context, string) ([]opt.Completion, string)) {
	x.input.TabComplete(
		tabComplete,
		config.Ui.CompletionDelay,
		config.Ui.CompletionMinChars,
		&config.Binds.Global.CompleteKey,
	)
}

func NewPrompt(prompt string, commit func(text string),
	tabcomplete func(ctx context.Context, cmd string) ([]opt.Completion, string),
) *ExLine {
	input := ui.NewTextInput("", config.Ui).Prompt(prompt)
	if config.Ui.CompletionPopovers {
		input.TabComplete(
			tabcomplete,
			config.Ui.CompletionDelay,
			config.Ui.CompletionMinChars,
			&config.Binds.Global.CompleteKey,
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

func (ex *ExLine) Event(event vaxis.Event) bool {
	if key, ok := event.(vaxis.Key); ok {
		switch {
		case key.Matches(vaxis.KeyEnter), key.Matches('j', vaxis.ModCtrl):
			cmd := ex.input.String()
			ex.input.Focus(false)
			ex.commit(cmd)
			ex.finish()
		case key.Matches(vaxis.KeyUp):
			ex.input.Set(ex.cmdHistory.Prev())
			ex.Invalidate()
		case key.Matches(vaxis.KeyDown):
			ex.input.Set(ex.cmdHistory.Next())
			ex.Invalidate()
		case key.Matches(vaxis.KeyEsc), key.Matches('c', vaxis.ModCtrl):
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
