package ui

import (
	"math"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/go-opt/v2"
	"git.sr.ht/~rockorager/vaxis"
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
	text              []vaxis.Character
	change            []func(ti *TextInput)
	focusLost         []func(ti *TextInput)
	tabcomplete       func(s string) ([]opt.Completion, string)
	completions       []opt.Completion
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
	chars := vaxis.Characters(text)
	return &TextInput{
		cells:    -1,
		text:     chars,
		index:    len(chars),
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
	tabcomplete func(s string) ([]opt.Completion, string),
	d time.Duration, minChars int, key *config.KeyStroke,
) *TextInput {
	ti.tabcomplete = tabcomplete
	ti.completeDelay = d
	ti.completeMinChars = minChars
	ti.completeKey = key
	return ti
}

func (ti *TextInput) String() string {
	return charactersToString(ti.text)
}

func (ti *TextInput) StringLeft() string {
	if ti.index > len(ti.text) {
		ti.index = len(ti.text)
	}
	left := ti.text[:ti.index]
	return charactersToString(left)
}

func (ti *TextInput) StringRight() string {
	if ti.index >= len(ti.text) {
		return ""
	}
	right := ti.text[ti.index:]
	return charactersToString(right)
}

func charactersToString(chars []vaxis.Character) string {
	buf := strings.Builder{}
	for _, ch := range chars {
		buf.WriteString(ch.Grapheme)
	}
	return buf.String()
}

func (ti *TextInput) Set(value string) *TextInput {
	ti.text = vaxis.Characters(value)
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
		cells := len(ti.text)
		ctx.Fill(x, 0, cells, 1, '*', defaultStyle)
	} else {
		ctx.Printf(0, 0, defaultStyle, "%s%s", ti.prompt, charactersToString(text))
	}
	cells := runewidth.StringWidth(charactersToString(text[:sindex]) + ti.prompt)
	if ti.focus {
		ctx.SetCursor(cells, 0, vaxis.CursorDefault)
		ti.drawPopover(ctx)
	}
}

func (ti *TextInput) drawPopover(ctx *Context) {
	if len(ti.completions) == 0 {
		return
	}

	valWidth := 0
	descWidth := 0
	for _, c := range ti.completions {
		valWidth = max(valWidth, runewidth.StringWidth(unquote(c.Value)))
		descWidth = max(descWidth, runewidth.StringWidth(c.Description))
	}
	descWidth = min(descWidth, 80)
	// one space padding
	width := 1 + valWidth
	if descWidth != 0 {
		// two spaces padding + parentheses
		width += 2 + descWidth + 2
	}
	// one space padding + gutter
	width += 2

	cmp := &completions{ti: ti, valWidth: valWidth, descWidth: descWidth}
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

func (ti *TextInput) MouseEvent(localX int, localY int, event vaxis.Event) {
	if event, ok := event.(vaxis.Mouse); ok {
		if event.Button == vaxis.MouseLeftButton {
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
		cells := runewidth.StringWidth(charactersToString(ti.text[:ti.index]))
		ti.ctx.SetCursor(cells+1, 0, vaxis.CursorDefault)
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

func (ti *TextInput) insert(ch vaxis.Character) {
	left := ti.text[:ti.index]
	right := ti.text[ti.index:]
	ti.text = append(left, append([]vaxis.Character{ch}, right...)...) //nolint:gocritic // intentional append to different slice
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
	for i >= 0 && ti.text[i].Grapheme == " " {
		i--
	}
	if i >= 0 && strings.Contains(separators, ti.text[i].Grapheme) {
		for i >= 0 && strings.Contains(separators, ti.text[i].Grapheme) {
			i--
		}
	} else {
		separators += " "
		for i >= 0 && !strings.Contains(separators, ti.text[i].Grapheme) {
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
		ti.Set(ti.prefix + ti.completions[ti.completeIndex].Value + ti.StringRight())
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

func (ti *TextInput) Event(event vaxis.Event) bool {
	ti.Lock()
	defer ti.Unlock()
	if key, ok := event.(vaxis.Key); ok {
		c := ti.completeKey
		if c != nil && key.Matches(c.Key, c.Modifiers) {
			ti.showCompletions(true)
			return true
		}

		ti.invalidateCompletions()

		switch {
		case key.Matches(vaxis.KeyBackspace):
			ti.backspace()
		case key.Matches('d', vaxis.ModCtrl), key.Matches(vaxis.KeyDelete):
			ti.deleteChar()
		case key.Matches('b', vaxis.ModCtrl), key.Matches(vaxis.KeyLeft):
			if ti.index > 0 {
				ti.index--
				ti.ensureScroll()
				ti.Invalidate()
			}
		case key.Matches('f', vaxis.ModCtrl), key.Matches(vaxis.KeyRight):
			if ti.index < len(ti.text) {
				ti.index++
				ti.ensureScroll()
				ti.Invalidate()
			}
		case key.Matches('a', vaxis.ModCtrl), key.Matches(vaxis.KeyHome):
			ti.index = 0
			ti.ensureScroll()
			ti.Invalidate()
		case key.Matches('e', vaxis.ModCtrl), key.Matches(vaxis.KeyEnd):
			ti.index = len(ti.text)
			ti.ensureScroll()
			ti.Invalidate()
		case key.Matches('k', vaxis.ModCtrl):
			ti.deleteLineForward()
		case key.Matches('w', vaxis.ModCtrl):
			ti.deleteWord()
		case key.Matches('u', vaxis.ModCtrl):
			ti.deleteLineBackward()
		case key.Matches(vaxis.KeyEsc):
			ti.Invalidate()
		case key.Text != "":
			chars := vaxis.Characters(key.Text)
			for _, ch := range chars {
				ti.insert(ch)
			}
		}
	}
	return true
}

type completions struct {
	ti        *TextInput
	valWidth  int
	descWidth int
}

func unquote(s string) string {
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		s = strings.ReplaceAll(s[1:len(s)-1], `'"'"'`, "'")
	}
	return s
}

func (c *completions) Draw(ctx *Context) {
	bg := c.ti.uiConfig.GetStyle(config.STYLE_COMPLETION_DEFAULT)
	bgDesc := c.ti.uiConfig.GetStyle(config.STYLE_COMPLETION_DESCRIPTION)
	gutter := c.ti.uiConfig.GetStyle(config.STYLE_COMPLETION_GUTTER)
	pill := c.ti.uiConfig.GetStyle(config.STYLE_COMPLETION_PILL)
	sel := c.ti.uiConfig.GetStyleSelected(config.STYLE_COMPLETION_DEFAULT)
	selDesc := c.ti.uiConfig.GetStyleSelected(config.STYLE_COMPLETION_DESCRIPTION)

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
		val := runewidth.FillRight(unquote(opt.Value), c.valWidth)
		desc := opt.Description
		if desc != "" {
			if runewidth.StringWidth(desc) > c.descWidth {
				desc = runewidth.Truncate(desc, c.descWidth, "…")
			}
			desc = "  " + runewidth.FillRight("("+desc+")", c.descWidth+2)
		}
		if c.index() == idx {
			n := ctx.Printf(0, idx-startIdx, sel, " %s", val)
			ctx.Printf(n, idx-startIdx, selDesc, "%s ", desc)
		} else {
			n := ctx.Printf(0, idx-startIdx, bg, " %s", val)
			ctx.Printf(n, idx-startIdx, bgDesc, "%s ", desc)
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

func (c *completions) Event(e vaxis.Event) bool {
	if e, ok := e.(vaxis.Key); ok {
		k := c.ti.completeKey
		if k != nil && e.Matches(k.Key, k.Modifiers) {
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

		switch {
		case e.Matches('n', vaxis.ModCtrl), e.Matches(vaxis.KeyDown):
			c.next()
			return true
		case e.Matches(vaxis.KeyTab, vaxis.ModShift),
			e.Matches('p', vaxis.ModCtrl),
			e.Matches(vaxis.KeyUp):
			c.prev()
			return true
		case e.Matches(vaxis.KeyEnter):
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
	c.ti.index = len(vaxis.Characters(c.ti.prefix + stem))
}

func findStem(words []opt.Completion) string {
	if len(words) == 0 {
		return ""
	}
	if len(words) == 1 {
		return words[0].Value
	}
	var stem string
	stemLen := 1
	firstWord := []rune(words[0].Value)
	for {
		if len(firstWord) < stemLen {
			return stem
		}
		var r rune = firstWord[stemLen-1]
		for _, word := range words[1:] {
			runes := []rune(word.Value)
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
