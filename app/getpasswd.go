package app

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rockorager/vaxis"
)

type GetPasswd struct {
	callback func(string, error)
	title    string
	prompt   string
	input    *ui.TextInput
}

func NewGetPasswd(
	title string, prompt string, cb func(string, error),
) *GetPasswd {
	getpasswd := &GetPasswd{
		callback: cb,
		title:    title,
		prompt:   prompt,
		input:    ui.NewTextInput("", config.Ui()).Password(true).Prompt("Password: "),
	}
	getpasswd.input.Focus(true)
	return getpasswd
}

func (gp *GetPasswd) Draw(ctx *ui.Context) {
	defaultStyle := config.Ui().GetStyle(config.STYLE_DEFAULT)
	titleStyle := config.Ui().GetStyle(config.STYLE_TITLE)

	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', defaultStyle)
	ctx.Fill(0, 0, ctx.Width(), 1, ' ', titleStyle)
	ctx.Printf(1, 0, titleStyle, "%s", gp.title)
	ctx.Printf(1, 1, defaultStyle, "%s", gp.prompt)
	gp.input.Draw(ctx.Subcontext(1, 3, ctx.Width()-2, 1))
}

func (gp *GetPasswd) Invalidate() {
	ui.Invalidate()
}

func (gp *GetPasswd) Event(event vaxis.Event) bool {
	switch event := event.(type) {
	case vaxis.Key:
		switch {
		case event.Matches(vaxis.KeyEnter):
			gp.input.Focus(false)
			gp.callback(gp.input.String(), nil)
		case event.Matches(vaxis.KeyEsc):
			gp.input.Focus(false)
			gp.callback("", fmt.Errorf("no password provided"))
		default:
			gp.input.Event(event)
		}
	default:
		gp.input.Event(event)
	}
	return true
}

func (gp *GetPasswd) Focus(f bool) {
	// Who cares
}
