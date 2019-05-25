package ui

import (
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

// TODO: Attach history and tab completion providers
// TODO: scrolling

type TextInput struct {
	Invalidatable
	cells    int
	ctx      *Context
	focus    bool
	index    int
	password bool
	prompt   string
	scroll   int
	text     []rune
	change   []func(ti *TextInput)
}

// Creates a new TextInput. TextInputs will render a "textbox" in the entire
// context they're given, and process keypresses to build a string from user
// input.
func NewTextInput(text string) *TextInput {
	return &TextInput{
		cells: -1,
		text:  []rune(text),
		index: len([]rune(text)),
	}
}

func (ti *TextInput) Password(password bool) *TextInput {
	ti.password = password
	return ti
}

func (ti *TextInput) Prompt(prompt string) *TextInput {
	ti.prompt = prompt
	return ti
}

func (ti *TextInput) String() string {
	return string(ti.text)
}

func (ti *TextInput) Set(value string) {
	ti.text = []rune(value)
	ti.index = len(ti.text)
}

func (ti *TextInput) Invalidate() {
	ti.DoInvalidate(ti)
}

func (ti *TextInput) Draw(ctx *Context) {
	scroll := ti.scroll
	if !ti.focus {
		scroll = 0
	}
	ti.ctx = ctx // gross
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)

	text := string(ti.text[scroll:])
	sindex := ti.index - scroll
	if ti.password {
		x := ctx.Printf(0, 0, tcell.StyleDefault, "%s", ti.prompt)
		cells := runewidth.StringWidth(string(text))
		ctx.Fill(x, 0, cells, 1, '*', tcell.StyleDefault)
	} else {
		ctx.Printf(0, 0, tcell.StyleDefault, "%s%s", ti.prompt, text)
	}
	cells := runewidth.StringWidth(text[:sindex] + ti.prompt)
	if ti.focus {
		ctx.SetCursor(cells, 0)
	}
}

func (ti *TextInput) Focus(focus bool) {
	ti.focus = focus
	if focus && ti.ctx != nil {
		cells := runewidth.StringWidth(string(ti.text[:ti.index]))
		ti.ctx.SetCursor(cells+1, 0)
	} else if !focus && ti.ctx != nil {
		ti.ctx.HideCursor()
	}
}

func (ti *TextInput) ensureScroll() {
	if ti.ctx == nil {
		return
	}
	// God why am I this lazy
	for ti.index-ti.scroll >= ti.ctx.Width() {
		ti.scroll++
	}
	for ti.index-ti.scroll < 0 {
		ti.scroll--
	}
}

func (ti *TextInput) insert(ch rune) {
	left := ti.text[:ti.index]
	right := ti.text[ti.index:]
	ti.text = append(left, append([]rune{ch}, right...)...)
	ti.index++
	ti.ensureScroll()
	ti.Invalidate()
	ti.onChange()
}

func (ti *TextInput) deleteWord() {
	// TODO: Break on any of / " '
	if len(ti.text) == 0 {
		return
	}
	i := ti.index - 1
	if ti.text[i] == ' ' {
		i--
	}
	for ; i >= 0; i-- {
		if ti.text[i] == ' ' {
			break
		}
	}
	ti.text = append(ti.text[:i+1], ti.text[ti.index:]...)
	ti.index = i + 1
	ti.ensureScroll()
	ti.Invalidate()
	ti.onChange()
}

func (ti *TextInput) deleteChar() {
	if len(ti.text) > 0 && ti.index != len(ti.text) {
		ti.text = append(ti.text[:ti.index], ti.text[ti.index+1:]...)
		ti.ensureScroll()
		ti.Invalidate()
		ti.onChange()
	}
}

func (ti *TextInput) backspace() {
	if len(ti.text) > 0 && ti.index != 0 {
		ti.text = append(ti.text[:ti.index-1], ti.text[ti.index:]...)
		ti.index--
		ti.ensureScroll()
		ti.Invalidate()
		ti.onChange()
	}
}

func (ti *TextInput) onChange() {
	for _, change := range ti.change {
		change(ti)
	}
}

func (ti *TextInput) OnChange(onChange func(ti *TextInput)) {
	ti.change = append(ti.change, onChange)
}

func (ti *TextInput) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			ti.backspace()
		case tcell.KeyCtrlD, tcell.KeyDelete:
			ti.deleteChar()
		case tcell.KeyCtrlB, tcell.KeyLeft:
			if ti.index > 0 {
				ti.index--
				ti.ensureScroll()
				ti.Invalidate()
			}
		case tcell.KeyCtrlF, tcell.KeyRight:
			if ti.index < len(ti.text) {
				ti.index++
				ti.ensureScroll()
				ti.Invalidate()
			}
		case tcell.KeyCtrlA, tcell.KeyHome:
			ti.index = 0
			ti.ensureScroll()
			ti.Invalidate()
		case tcell.KeyCtrlE, tcell.KeyEnd:
			ti.index = len(ti.text)
			ti.ensureScroll()
			ti.Invalidate()
		case tcell.KeyCtrlW:
			ti.deleteWord()
		case tcell.KeyRune:
			ti.insert(event.Rune())
		}
	}
	return true
}
