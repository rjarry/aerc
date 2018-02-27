package ui

import (
	tb "github.com/nsf/termbox-go"
)

// TODO: history
// TODO: tab completion
// TODO: commit
// TODO: cancel (via esc/ctrl+c)
// TODO: scrolling

type ExLine struct {
	command *string
	commit  func(cmd *string)
	index   int
	scroll  int

	onInvalidate func(d Drawable)
}

func NewExLine() *ExLine {
	cmd := ""
	return &ExLine{command: &cmd}
}

func (ex *ExLine) OnInvalidate(onInvalidate func(d Drawable)) {
	ex.onInvalidate = onInvalidate
}

func (ex *ExLine) Invalidate() {
	if ex.onInvalidate != nil {
		ex.onInvalidate(ex)
	}
}

func (ex *ExLine) Draw(ctx *Context) {
	cell := tb.Cell{
		Fg: tb.ColorDefault,
		Bg: tb.ColorDefault,
		Ch: ' ',
	}
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), cell)
	ctx.Printf(0, 0, cell, ":%s", *ex.command)
	tb.SetCursor(ctx.X()+ex.index-ex.scroll+1, ctx.Y())
}

func (ex *ExLine) insert(ch rune) {
	newCmd := (*ex.command)[:ex.index] + string(ch) + (*ex.command)[ex.index:]
	ex.command = &newCmd
	ex.index++
	ex.Invalidate()
}

func (ex *ExLine) deleteWord() {
	// TODO: Break on any of / " '
	if len(*ex.command) == 0 {
		return
	}
	i := ex.index - 1
	if (*ex.command)[i] == ' ' {
		i--
	}
	for ; i >= 0; i-- {
		if (*ex.command)[i] == ' ' {
			break
		}
	}
	newCmd := (*ex.command)[:i+1] + (*ex.command)[ex.index:]
	ex.command = &newCmd
	ex.index = i + 1
	ex.Invalidate()
}

func (ex *ExLine) deleteChar() {
	if len(*ex.command) > 0 && ex.index != len(*ex.command) {
		newCmd := (*ex.command)[:ex.index] + (*ex.command)[ex.index+1:]
		ex.command = &newCmd
		ex.Invalidate()
	}
}

func (ex *ExLine) backspace() {
	if len(*ex.command) > 0 && ex.index != 0 {
		newCmd := (*ex.command)[:ex.index-1] + (*ex.command)[ex.index:]
		ex.command = &newCmd
		ex.index--
		ex.Invalidate()
	}
}

func (ex *ExLine) Event(event tb.Event) bool {
	switch event.Type {
	case tb.EventKey:
		switch event.Key {
		case tb.KeySpace:
			ex.insert(' ')
		case tb.KeyBackspace, tb.KeyBackspace2:
			ex.backspace()
		case tb.KeyCtrlD, tb.KeyDelete:
			ex.deleteChar()
		case tb.KeyCtrlB, tb.KeyArrowLeft:
			if ex.index > 0 {
				ex.index--
				ex.Invalidate()
			}
		case tb.KeyCtrlF, tb.KeyArrowRight:
			if ex.index < len(*ex.command) {
				ex.index++
				ex.Invalidate()
			}
		case tb.KeyCtrlA, tb.KeyHome:
			ex.index = 0
			ex.Invalidate()
		case tb.KeyCtrlE, tb.KeyEnd:
			ex.index = len(*ex.command)
			ex.Invalidate()
		case tb.KeyCtrlW:
			ex.deleteWord()
		default:
			if event.Ch != 0 {
				ex.insert(event.Ch)
			}
		}
	}
	return true
}
