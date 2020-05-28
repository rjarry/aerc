package widgets

import (
	"fmt"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/lib/ui"
)

type GetPasswd struct {
	ui.Invalidatable
	callback func(string, error)
	title    string
	prompt   string
	input    *ui.TextInput
}

func NewGetPasswd(title string, prompt string, cb func(string, error)) *GetPasswd {
	getpasswd := &GetPasswd{
		callback: cb,
		title:    title,
		prompt:   prompt,
		input:    ui.NewTextInput("").Password(true).Prompt("Password: "),
	}
	getpasswd.input.OnInvalidate(func(_ ui.Drawable) {
		getpasswd.Invalidate()
	})
	getpasswd.input.Focus(true)
	return getpasswd
}

func (gp *GetPasswd) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	ctx.Fill(0, 0, ctx.Width(), 1, ' ', tcell.StyleDefault.Reverse(true))
	ctx.Printf(1, 0, tcell.StyleDefault.Reverse(true), "%s", gp.title)
	ctx.Printf(1, 1, tcell.StyleDefault, gp.prompt)
	gp.input.Draw(ctx.Subcontext(1, 3, ctx.Width()-2, 1))
}

func (gp *GetPasswd) Invalidate() {
	gp.DoInvalidate(gp)
}

func (gp *GetPasswd) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyEnter:
			gp.input.Focus(false)
			gp.callback(gp.input.String(), nil)
		case tcell.KeyEsc:
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
