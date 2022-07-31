package widgets

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type GetPasswd struct {
	ui.Invalidatable
	callback func(string, error)
	title    string
	prompt   string
	input    *ui.TextInput
	conf     *config.AercConfig
}

func NewGetPasswd(title string, prompt string, conf *config.AercConfig,
	cb func(string, error),
) *GetPasswd {
	getpasswd := &GetPasswd{
		callback: cb,
		title:    title,
		prompt:   prompt,
		conf:     conf,
		input:    ui.NewTextInput("", &conf.Ui).Password(true).Prompt("Password: "),
	}
	getpasswd.input.OnInvalidate(func(_ ui.Drawable) {
		getpasswd.Invalidate()
	})
	getpasswd.input.Focus(true)
	return getpasswd
}

func (gp *GetPasswd) Draw(ctx *ui.Context) {
	defaultStyle := gp.conf.Ui.GetStyle(config.STYLE_DEFAULT)
	titleStyle := gp.conf.Ui.GetStyle(config.STYLE_TITLE)

	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', defaultStyle)
	ctx.Fill(0, 0, ctx.Width(), 1, ' ', titleStyle)
	ctx.Printf(1, 0, titleStyle, "%s", gp.title)
	ctx.Printf(1, 1, defaultStyle, gp.prompt)
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
