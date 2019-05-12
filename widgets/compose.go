package widgets

import (
	"os/exec"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type headerEditor struct {
	ui.Invalidatable
	name  string
	input *ui.TextInput
}

type Composer struct {
	headers struct {
		from    *headerEditor
		subject *headerEditor
		to      *headerEditor
	}

	editor *Terminal
	grid   *ui.Grid

	focusable []ui.DrawableInteractive
	focused   int
}

// TODO: Let caller configure headers, initial body (for replies), etc
func NewComposer() *Composer {
	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 3},
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})

	// TODO: let user specify extra headers to edit by default
	headers := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 1}, // To/From
		{ui.SIZE_EXACT, 1}, // Subject
		{ui.SIZE_EXACT, 1}, // [spacer]
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_WEIGHT, 1},
	})

	headers.AddChild(newHeaderEditor("To", "Simon Ser <contact@emersion.fr>")).At(0, 0)
	headers.AddChild(newHeaderEditor("From", "Drew DeVault <sir@cmpwn.com>")).At(0, 1)
	headers.AddChild(newHeaderEditor("Subject", "Re: [PATCH RFC aerc2] widgets: fix StatusLine race")).At(1, 0).Span(1, 2)
	headers.AddChild(ui.NewFill(' ')).At(2, 0).Span(1, 2)

	// TODO: built-in config option, $EDITOR, then vi, in that order
	// TODO: temp file
	editor := exec.Command("vim")
	term, _ := NewTerminal(editor)

	grid.AddChild(headers).At(0, 0)
	grid.AddChild(term).At(1, 0)

	return &Composer{
		grid:    grid,
		editor:  term,
		focused: 0,
		focusable: []ui.DrawableInteractive{
			term,
		},
	}
}

func (c *Composer) Draw(ctx *ui.Context) {
	c.grid.Draw(ctx)
}

func (c *Composer) Invalidate() {
	c.grid.Invalidate()
}

func (c *Composer) OnInvalidate(fn func(d ui.Drawable)) {
	c.grid.OnInvalidate(func(_ ui.Drawable) {
		fn(c)
	})
}

// TODO: Focus various fields separately
// TODO: Consider having a different set of keybindings for a focused and
// unfocused terminal?
func (c *Composer) Event(event tcell.Event) bool {
	if c.editor != nil {
		return c.editor.Event(event)
	}
	return false
}

func (c *Composer) Focus(focus bool) {
	if c.editor != nil {
		c.editor.Focus(focus)
	}
}

func newHeaderEditor(name string, value string) *headerEditor {
	// TODO: Set default vaule to something sane, I guess
	return &headerEditor{
		input: ui.NewTextInput(value),
		name:  name,
	}
}

func (he *headerEditor) Draw(ctx *ui.Context) {
	name := he.name + " "
	size := runewidth.StringWidth(name)
	ctx.Fill(0, 0, size, ctx.Height(), ' ', tcell.StyleDefault)
	ctx.Printf(0, 0, tcell.StyleDefault.Bold(true), "%s", name)
	he.input.Draw(ctx.Subcontext(size, 0, ctx.Width()-size, 1))
}

func (he *headerEditor) Invalidate() {
	he.DoInvalidate(he)
}
