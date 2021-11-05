package widgets

import (
	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Selector struct {
	ui.Invalidatable
	chooser  bool
	focused  bool
	focus    int
	options  []string
	uiConfig config.UIConfig

	onChoose func(option string)
	onSelect func(option string)
}

func NewSelector(options []string, focus int, uiConfig config.UIConfig) *Selector {
	return &Selector{
		focus:    focus,
		options:  options,
		uiConfig: uiConfig,
	}
}

func (sel *Selector) Chooser(chooser bool) *Selector {
	sel.chooser = chooser
	return sel
}

func (sel *Selector) Invalidate() {
	sel.DoInvalidate(sel)
}

func (sel *Selector) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
		sel.uiConfig.GetStyle(config.STYLE_SELECTOR_DEFAULT))
	x := 2
	for i, option := range sel.options {
		style := sel.uiConfig.GetStyle(config.STYLE_SELECTOR_DEFAULT)
		if sel.focus == i {
			if sel.focused {
				style = sel.uiConfig.GetStyle(config.STYLE_SELECTOR_FOCUSED)
			} else if sel.chooser {
				style = sel.uiConfig.GetStyle(config.STYLE_SELECTOR_CHOOSER)
			}
		}
		x += ctx.Printf(x, 1, style, "[%s]", option)
		x += 5
	}
}

func (sel *Selector) OnChoose(fn func(option string)) *Selector {
	sel.onChoose = fn
	return sel
}

func (sel *Selector) OnSelect(fn func(option string)) *Selector {
	sel.onSelect = fn
	return sel
}

func (sel *Selector) Selected() string {
	return sel.options[sel.focus]
}

func (sel *Selector) Focus(focus bool) {
	sel.focused = focus
	sel.Invalidate()
}

func (sel *Selector) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyCtrlH:
			fallthrough
		case tcell.KeyLeft:
			if sel.focus > 0 {
				sel.focus--
				sel.Invalidate()
			}
			if sel.onSelect != nil {
				sel.onSelect(sel.Selected())
			}
		case tcell.KeyCtrlL:
			fallthrough
		case tcell.KeyRight:
			if sel.focus < len(sel.options)-1 {
				sel.focus++
				sel.Invalidate()
			}
			if sel.onSelect != nil {
				sel.onSelect(sel.Selected())
			}
		case tcell.KeyEnter:
			if sel.onChoose != nil {
				sel.onChoose(sel.Selected())
			}
		}
	}
	return false
}
