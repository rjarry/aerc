package widgets

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

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
	defaultSelectorStyle := sel.uiConfig.GetStyle(config.STYLE_SELECTOR_DEFAULT)
	w, h := ctx.Width(), ctx.Height()
	ctx.Fill(0, 0, w, h, ' ', defaultSelectorStyle)

	if w < 5 || h < 1 {
		// if width and height are that small, don't even try to draw
		// something
		return
	}

	y := 1
	if h == 1 {
		y = 0
	}

	format := "[%s]"

	calculateWidth := func(space int) int {
		neededWidth := 2
		for i, option := range sel.options {
			neededWidth += runewidth.StringWidth(fmt.Sprintf(format, option))
			if i < len(sel.options)-1 {
				neededWidth += space
			}
		}
		return neededWidth - space
	}

	space := 5
	for ; space > 0; space-- {
		if w > calculateWidth(space) {
			break
		}
	}

	x := 2
	for i, option := range sel.options {
		style := defaultSelectorStyle
		if sel.focus == i {
			if sel.focused {
				style = sel.uiConfig.GetStyle(config.STYLE_SELECTOR_FOCUSED)
			} else if sel.chooser {
				style = sel.uiConfig.GetStyle(config.STYLE_SELECTOR_CHOOSER)
			}
		}

		if space == 0 {
			if sel.focus == i {
				leftArrow, rightArrow := ' ', ' '
				if i > 0 {
					leftArrow = '❮'
				}
				if i < len(sel.options)-1 {
					rightArrow = '❯'
				}

				s := runewidth.Truncate(option,
					w-runewidth.RuneWidth(leftArrow)-runewidth.RuneWidth(rightArrow)-runewidth.StringWidth(fmt.Sprintf(format, "")),
					"…")

				nextPos := 0
				nextPos += ctx.Printf(nextPos, y, defaultSelectorStyle, "%c", leftArrow)
				nextPos += ctx.Printf(nextPos, y, style, format, s)
				ctx.Printf(nextPos, y, defaultSelectorStyle, "%c", rightArrow)
			}
		} else {
			x += ctx.Printf(x, y, style, format, option)
			x += space
		}
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

var ErrNoOptionSelected = fmt.Errorf("no option selected")

type SelectorDialog struct {
	ui.Invalidatable
	callback func(string, error)
	title    string
	prompt   string
	uiConfig config.UIConfig
	selector *Selector
}

func NewSelectorDialog(title string, prompt string, options []string, focus int,
	uiConfig config.UIConfig, cb func(string, error)) *SelectorDialog {
	sd := &SelectorDialog{
		callback: cb,
		title:    title,
		prompt:   prompt,
		uiConfig: uiConfig,
		selector: NewSelector(options, focus, uiConfig).Chooser(true),
	}
	sd.selector.OnInvalidate(func(_ ui.Drawable) {
		sd.Invalidate()
	})
	sd.selector.Focus(true)
	return sd
}

func (gp *SelectorDialog) Draw(ctx *ui.Context) {
	defaultStyle := gp.uiConfig.GetStyle(config.STYLE_DEFAULT)
	titleStyle := gp.uiConfig.GetStyle(config.STYLE_TITLE)

	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', defaultStyle)
	ctx.Fill(0, 0, ctx.Width(), 1, ' ', titleStyle)
	ctx.Printf(1, 0, titleStyle, "%s", gp.title)
	ctx.Printf(1, 1, defaultStyle, gp.prompt)
	gp.selector.Draw(ctx.Subcontext(1, 3, ctx.Width()-2, 1))
}

func (gp *SelectorDialog) Invalidate() {
	gp.DoInvalidate(gp)
}

func (gp *SelectorDialog) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyEnter:
			gp.selector.Focus(false)
			gp.callback(gp.selector.Selected(), nil)
		case tcell.KeyEsc:
			gp.selector.Focus(false)
			gp.callback("", ErrNoOptionSelected)
		default:
			gp.selector.Event(event)
		}
	default:
		gp.selector.Event(event)
	}
	return true
}

func (gp *SelectorDialog) Focus(f bool) {
	gp.selector.Focus(f)
}
