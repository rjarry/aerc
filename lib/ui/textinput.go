package ui

import (
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/log"
)

// TODO: Attach history providers

type TextInput struct {
	sync.Mutex
	cells             int
	ctx               *Context
	focus             bool
	index             int
	password          bool
	prompt            string
	scroll            int
	text              []rune
	change            []func(ti *TextInput)
	focusLost         []func(ti *TextInput)
	tabcomplete       func(s string) ([]string, string)
	completions       []string
	prefix            string
	completeIndex     int
	completeDelay     time.Duration
	completeDebouncer *time.Timer
	completeMinChars  int
	completeKey       *config.KeyStroke
	uiConfig          *config.UIConfig
}

// Creates a new TextInput. TextInputs will render a "textbox" in the entire
// context they're given, and process keypresses to build a string from user
// input.
func NewTextInput(text string, ui *config.UIConfig) *TextInput {
	return &TextInput{
		cells:    -1,
		text:     []rune(text),
		index:    len([]rune(text)),
		uiConfig: ui,
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

func (ti *TextInput) TabComplete(
	tabcomplete func(s string) ([]string, string),
	d time.Duration, minChars int, key *config.KeyStroke,
) *TextInput {
	ti.tabcomplete = tabcomplete
	ti.completeDelay = d
	ti.completeMinChars = minChars
	ti.completeKey = key
	return ti
}

func (ti *TextInput) String() string {
	return string(ti.text)
}

func (ti *TextInput) StringLeft() string {
	for ti.index > len(ti.text) {
		ti.index = len(ti.text)
	}
	return string(ti.text[:ti.index])
}

func (ti *TextInput) StringRight() string {
	return string(ti.text[ti.index:])
}

func (ti *TextInput) Set(value string) *TextInput {
	ti.text = []rune(value)
	ti.index = len(ti.text)
	ti.scroll = 0
	return ti
}

func (ti *TextInput) Invalidate() {
	Invalidate()
}

func (ti *TextInput) Draw(ctx *Context) {
	scroll := 0
	if ti.focus {
		ti.ensureScroll()
		scroll = ti.scroll
	}
	ti.ctx = ctx // gross

	defaultStyle := ti.uiConfig.GetStyle(config.STYLE_DEFAULT)
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', defaultStyle)

	text := ti.text[scroll:]
	sindex := ti.index - scroll
	if ti.password {
		x := ctx.Printf(0, 0, defaultStyle, "%s", ti.prompt)
		cells := runewidth.StringWidth(string(text))
		ctx.Fill(x, 0, cells, 1, '*', defaultStyle)
	} else {
		ctx.Printf(0, 0, defaultStyle, "%s%s", ti.prompt, string(text))
	}
	cells := runewidth.StringWidth(string(text[:sindex]) + ti.prompt)
	if ti.focus {
		ctx.SetCursor(cells, 0)
		ti.drawPopover(ctx)
	}
}

func (ti *TextInput) drawPopover(ctx *Context) {
	if len(ti.completions) == 0 {
		return
	}
	cmp := &completions{ti: ti}
	width := maxLen(ti.completions) + 3
	height := len(ti.completions)

	pos := len(ti.prefix) - ti.scroll
	if pos+width > ctx.Width() {
		pos = ctx.Width() - width
	}
	if pos < 0 {
		pos = 0
	}

	ctx.Popover(pos, 0, width, height, cmp)
}

func (ti *TextInput) MouseEvent(localX int, localY int, event tcell.Event) {
	if event, ok := event.(*tcell.EventMouse); ok {
		if event.Buttons() == tcell.Button1 {
			if localX >= len(ti.prompt)+1 && localX <= len(ti.text[ti.scroll:])+len(ti.prompt)+1 {
				ti.index = localX - len(ti.prompt) - 1
				ti.ensureScroll()
				ti.Invalidate()
			}
		}
	}
}

func (ti *TextInput) Focus(focus bool) {
	if ti.focus && !focus {
		ti.onFocusLost()
	}
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
	w := ti.ctx.Width() - len(ti.prompt)
	if ti.index >= ti.scroll+w {
		ti.scroll = ti.index - w + 1
	}
	if ti.index < ti.scroll {
		ti.scroll = ti.index
	}
}

func (ti *TextInput) insert(ch rune) {
	left := ti.text[:ti.index]
	right := ti.text[ti.index:]
	ti.text = append(left, append([]rune{ch}, right...)...) //nolint:gocritic // intentional append to different slice
	ti.index++
	ti.ensureScroll()
	ti.Invalidate()
	ti.onChange()
}

func (ti *TextInput) deleteWord() {
	if len(ti.text) == 0 || ti.index <= 0 {
		return
	}
	separators := "/'\""
	i := ti.index - 1
	for i >= 0 && ti.text[i] == ' ' {
		i--
	}
	if i >= 0 && strings.ContainsRune(separators, ti.text[i]) {
		for i >= 0 && strings.ContainsRune(separators, ti.text[i]) {
			i--
		}
	} else {
		separators += " "
		for i >= 0 && !strings.ContainsRune(separators, ti.text[i]) {
			i--
		}
	}
	ti.text = append(ti.text[:i+1], ti.text[ti.index:]...)
	ti.index = i + 1
	ti.ensureScroll()
	ti.Invalidate()
	ti.onChange()
}

func (ti *TextInput) deleteLineForward() {
	if len(ti.text) == 0 || len(ti.text) == ti.index {
		return
	}

	ti.text = ti.text[:ti.index]
	ti.ensureScroll()
	ti.Invalidate()
	ti.onChange()
}

func (ti *TextInput) deleteLineBackward() {
	if len(ti.text) == 0 || ti.index == 0 {
		return
	}

	ti.text = ti.text[ti.index:]
	ti.index = 0
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

func (ti *TextInput) executeCompletion() {
	if len(ti.completions) > 0 {
		ti.Set(ti.prefix + ti.completions[ti.completeIndex] + ti.StringRight())
	}
}

func (ti *TextInput) invalidateCompletions() {
	ti.completions = nil
}

func (ti *TextInput) onChange() {
	ti.updateCompletions()
	for _, change := range ti.change {
		change(ti)
	}
}

func (ti *TextInput) onFocusLost() {
	for _, focusLost := range ti.focusLost {
		focusLost(ti)
	}
}

func (ti *TextInput) updateCompletions() {
	if ti.tabcomplete == nil {
		// no completer
		return
	}
	if ti.completeMinChars == config.MANUAL_COMPLETE {
		// only manually triggered completion
		return
	}
	if ti.completeDebouncer == nil {
		ti.completeDebouncer = time.AfterFunc(ti.completeDelay, func() {
			defer log.PanicHandler()
			ti.Lock()
			if len(ti.StringLeft()) >= ti.completeMinChars {
				ti.showCompletions(false)
			}
			ti.Unlock()
		})
	} else {
		ti.completeDebouncer.Stop()
		ti.completeDebouncer.Reset(ti.completeDelay)
	}
}

func (ti *TextInput) showCompletions(explicit bool) {
	if ti.tabcomplete == nil {
		// no completer
		return
	}
	ti.completions, ti.prefix = ti.tabcomplete(ti.StringLeft())

	if explicit && len(ti.completions) == 1 {
		// automatically accept if there is only one choice
		ti.completeIndex = 0
		ti.executeCompletion()
		ti.invalidateCompletions()
	} else {
		ti.completeIndex = -1
	}
	Invalidate()
}

func (ti *TextInput) OnChange(onChange func(ti *TextInput)) {
	ti.change = append(ti.change, onChange)
}

func (ti *TextInput) OnFocusLost(onFocusLost func(ti *TextInput)) {
	ti.focusLost = append(ti.focusLost, onFocusLost)
}

func (ti *TextInput) Event(event tcell.Event) bool {
	ti.Lock()
	defer ti.Unlock()
	if event, ok := event.(*tcell.EventKey); ok {
		c := ti.completeKey
		if c != nil && c.Key == event.Key() && c.Modifiers == event.Modifiers() {
			ti.showCompletions(true)
			return true
		}

		ti.invalidateCompletions()

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
		case tcell.KeyCtrlK:
			ti.deleteLineForward()
		case tcell.KeyCtrlW:
			ti.deleteWord()
		case tcell.KeyCtrlU:
			ti.deleteLineBackward()
		case tcell.KeyESC:
			ti.Invalidate()
		case tcell.KeyRune:
			ti.insert(event.Rune())
		}
	}
	return true
}

type completions struct {
	ti *TextInput
}

func unquote(s string) string {
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		s = strings.ReplaceAll(s[1:len(s)-1], `'"'"'`, "'")
	}
	return s
}

func maxLen(ss []string) int {
	max := 0
	for _, s := range ss {
		l := runewidth.StringWidth(unquote(s))
		if l > max {
			max = l
		}
	}
	return max
}

func (c *completions) Draw(ctx *Context) {
	bg := c.ti.uiConfig.GetStyle(config.STYLE_COMPLETION_DEFAULT)
	gutter := c.ti.uiConfig.GetStyle(config.STYLE_COMPLETION_GUTTER)
	pill := c.ti.uiConfig.GetStyle(config.STYLE_COMPLETION_PILL)
	sel := c.ti.uiConfig.GetStyleSelected(config.STYLE_COMPLETION_DEFAULT)

	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', bg)

	numVisible := ctx.Height()
	startIdx := 0
	if len(c.ti.completions) > numVisible && c.index()+1 > numVisible {
		startIdx = c.index() - (numVisible - 1)
	}
	endIdx := startIdx + numVisible - 1

	for idx, opt := range c.ti.completions {
		if idx < startIdx {
			continue
		}
		if idx > endIdx {
			continue
		}
		if c.index() == idx {
			ctx.Fill(0, idx-startIdx, ctx.Width(), 1, ' ', sel)
			ctx.Printf(0, idx-startIdx, sel, " %s ", unquote(opt))
		} else {
			ctx.Printf(0, idx-startIdx, bg, " %s ", unquote(opt))
		}
	}

	percentVisible := float64(numVisible) / float64(len(c.ti.completions))
	if percentVisible >= 1.0 {
		return
	}

	// gutter
	ctx.Fill(ctx.Width()-1, 0, 1, ctx.Height(), ' ', gutter)

	pillSize := int(math.Ceil(float64(ctx.Height()) * percentVisible))
	percentScrolled := float64(startIdx) / float64(len(c.ti.completions))
	pillOffset := int(math.Floor(float64(ctx.Height()) * percentScrolled))
	ctx.Fill(ctx.Width()-1, pillOffset, 1, pillSize, ' ', pill)
}

func (c *completions) index() int {
	return c.ti.completeIndex
}

func (c *completions) next() {
	index := c.index()
	index++
	if index >= len(c.ti.completions) {
		index = -1
	}
	c.ti.completeIndex = index
	Invalidate()
}

func (c *completions) prev() {
	index := c.index()
	index--
	if index < -1 {
		index = len(c.ti.completions) - 1
	}
	c.ti.completeIndex = index
	Invalidate()
}

func (c *completions) exec() {
	c.ti.executeCompletion()
	c.ti.invalidateCompletions()
	Invalidate()
}

func (c *completions) Event(e tcell.Event) bool {
	if e, ok := e.(*tcell.EventKey); ok {
		k := c.ti.completeKey
		if k != nil && k.Key == e.Key() && k.Modifiers == e.Modifiers() {
			if len(c.ti.completions) == 1 {
				c.ti.completeIndex = 0
				c.exec()
			} else {
				stem := findStem(c.ti.completions)
				if c.needsStem(stem) {
					c.stem(stem)
				}
				c.next()
			}
			return true
		}

		switch e.Key() {
		case tcell.KeyCtrlN, tcell.KeyDown:
			c.next()
			return true
		case tcell.KeyBacktab, tcell.KeyCtrlP, tcell.KeyUp:
			c.prev()
			return true
		case tcell.KeyEnter:
			if c.index() >= 0 {
				c.exec()
				return true
			}
		}
	}
	return false
}

func (c *completions) needsStem(stem string) bool {
	if stem == "" || c.index() >= 0 {
		return false
	}
	if len(stem)+len(c.ti.prefix) > len(c.ti.StringLeft()) {
		return true
	}
	return false
}

func (c *completions) stem(stem string) {
	c.ti.Set(c.ti.prefix + stem + c.ti.StringRight())
	c.ti.index = runewidth.StringWidth(c.ti.prefix + stem)
}

func findStem(words []string) string {
	if len(words) == 0 {
		return ""
	}
	if len(words) == 1 {
		return words[0]
	}
	var stem string
	stemLen := 1
	firstWord := []rune(words[0])
	for {
		if len(firstWord) < stemLen {
			return stem
		}
		var r rune = firstWord[stemLen-1]
		for _, word := range words[1:] {
			runes := []rune(word)
			if len(runes) < stemLen {
				return stem
			}
			if runes[stemLen-1] != r {
				return stem
			}
		}
		stem += string(r)
		stemLen++
	}
}

func (c *completions) Focus(_ bool) {}

func (c *completions) Invalidate() {}
