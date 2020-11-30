package ui

import (
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc/config"
)

type Tabs struct {
	Tabs       []*Tab
	TabStrip   *TabStrip
	TabContent *TabContent
	Selected   int
	history    []int

	uiConfig *config.UIConfig

	onInvalidateStrip   func(d Drawable)
	onInvalidateContent func(d Drawable)

	parent   *Tabs
	CloseTab func(index int)
}

type Tab struct {
	Content        Drawable
	Name           string
	invalid        bool
	pinned         bool
	indexBeforePin int
}

type TabStrip Tabs
type TabContent Tabs

func NewTabs(uiConf *config.UIConfig) *Tabs {
	tabs := &Tabs{}
	tabs.uiConfig = uiConf
	tabs.TabStrip = (*TabStrip)(tabs)
	tabs.TabStrip.parent = tabs
	tabs.TabContent = (*TabContent)(tabs)
	tabs.TabContent.parent = tabs
	tabs.history = []int{}
	return tabs
}

func (tabs *Tabs) Add(content Drawable, name string) *Tab {
	tab := &Tab{
		Content: content,
		Name:    name,
	}
	tabs.Tabs = append(tabs.Tabs, tab)
	tabs.TabStrip.Invalidate()
	content.OnInvalidate(tabs.invalidateChild)
	return tab
}

func (tabs *Tabs) invalidateChild(d Drawable) {
	if tabs.Selected >= len(tabs.Tabs) {
		return
	}

	if tabs.Tabs[tabs.Selected].Content == d {
		if tabs.onInvalidateContent != nil {
			tabs.onInvalidateContent(tabs.TabContent)
		}
	}
}

func (tabs *Tabs) Remove(content Drawable) {
	indexToRemove := -1
	for i, tab := range tabs.Tabs {
		if tab.Content == content {
			tabs.Tabs = append(tabs.Tabs[:i], tabs.Tabs[i+1:]...)
			tabs.removeHistory(i)
			indexToRemove = i
			break
		}
	}
	if indexToRemove < 0 {
		return
	}
	// only pop the tab history if the closing tab is selected
	if indexToRemove == tabs.Selected {
		index, ok := tabs.popHistory()
		if ok {
			tabs.Select(index)
			interactive, ok := tabs.Tabs[tabs.Selected].Content.(Interactive)
			if ok {
				interactive.Focus(true)
			}
		}
	} else if indexToRemove < tabs.Selected {
		// selected tab is now one to the left of where it was
		tabs.Selected--
	}
	tabs.TabStrip.Invalidate()
}

func (tabs *Tabs) Replace(contentSrc Drawable, contentTarget Drawable, name string) {
	replaceTab := &Tab{
		Content: contentTarget,
		Name:    name,
	}
	for i, tab := range tabs.Tabs {
		if tab.Content == contentSrc {
			tabs.Tabs[i] = replaceTab
			tabs.Select(i)
			if c, ok := contentSrc.(io.Closer); ok {
				c.Close()
			}
			break
		}
	}
	tabs.TabStrip.Invalidate()
	contentTarget.OnInvalidate(tabs.invalidateChild)
}

func (tabs *Tabs) Select(index int) {
	if index >= len(tabs.Tabs) {
		index = len(tabs.Tabs) - 1
	}

	if tabs.Selected != index {
		// only push valid tabs onto the history
		if tabs.Selected < len(tabs.Tabs) {
			tabs.pushHistory(tabs.Selected)
		}
		tabs.Selected = index
		tabs.TabStrip.Invalidate()
		tabs.TabContent.Invalidate()
	}
}

func (tabs *Tabs) SelectPrevious() bool {
	index, ok := tabs.popHistory()
	if !ok {
		return false
	}
	tabs.Select(index)
	return true
}

func (tabs *Tabs) MoveTab(to int) {
	from := tabs.Selected

	if to < 0 {
		to = 0
	}

	if to >= len(tabs.Tabs) {
		to = len(tabs.Tabs) - 1
	}

	tab := tabs.Tabs[from]
	if to > from {
		copy(tabs.Tabs[from:to], tabs.Tabs[from+1:to+1])
		for i, h := range tabs.history {
			if h == from {
				tabs.history[i] = to
			}
			if h > from && h <= to {
				tabs.history[i] -= 1
			}
		}
	} else if from > to {
		copy(tabs.Tabs[to+1:from+1], tabs.Tabs[to:from])
		for i, h := range tabs.history {
			if h == from {
				tabs.history[i] = to
			}
			if h >= to && h < from {
				tabs.history[i] += 1
			}
		}
	} else {
		return
	}

	tabs.Tabs[to] = tab
	tabs.Selected = to
	tabs.TabStrip.Invalidate()
}

func (tabs *Tabs) PinTab() {
	if tabs.Tabs[tabs.Selected].pinned {
		return
	}

	pinEnd := len(tabs.Tabs)
	for i, t := range tabs.Tabs {
		if !t.pinned {
			pinEnd = i
			break
		}
	}

	for _, t := range tabs.Tabs {
		if t.pinned && t.indexBeforePin > tabs.Selected-pinEnd {
			t.indexBeforePin -= 1
		}
	}

	tabs.Tabs[tabs.Selected].pinned = true
	tabs.Tabs[tabs.Selected].indexBeforePin = tabs.Selected - pinEnd

	tabs.MoveTab(pinEnd)
}

func (tabs *Tabs) UnpinTab() {
	if !tabs.Tabs[tabs.Selected].pinned {
		return
	}

	pinEnd := len(tabs.Tabs)
	for i, t := range tabs.Tabs {
		if i != tabs.Selected && t.pinned && t.indexBeforePin > tabs.Tabs[tabs.Selected].indexBeforePin {
			t.indexBeforePin += 1
		}
		if !t.pinned {
			pinEnd = i
			break
		}
	}

	tabs.Tabs[tabs.Selected].pinned = false

	tabs.MoveTab(tabs.Tabs[tabs.Selected].indexBeforePin + pinEnd - 1)
}

func (tabs *Tabs) NextTab() {
	next := tabs.Selected + 1
	if next >= len(tabs.Tabs) {
		next = 0
	}
	tabs.Select(next)
}

func (tabs *Tabs) PrevTab() {
	next := tabs.Selected - 1
	if next < 0 {
		next = len(tabs.Tabs) - 1
	}
	tabs.Select(next)
}

func (tabs *Tabs) pushHistory(index int) {
	tabs.history = append(tabs.history, index)
}

func (tabs *Tabs) popHistory() (int, bool) {
	lastIdx := len(tabs.history) - 1
	if lastIdx < 0 {
		return 0, false
	}
	item := tabs.history[lastIdx]
	tabs.history = tabs.history[:lastIdx]
	return item, true
}

func (tabs *Tabs) removeHistory(index int) {
	newHist := make([]int, 0, len(tabs.history))
	for i, item := range tabs.history {
		if item == index {
			continue
		}
		if item > index {
			item = item - 1
		}
		// dedup
		if i > 0 && len(newHist) > 0 && item == newHist[len(newHist)-1] {
			continue
		}
		newHist = append(newHist, item)
	}
	tabs.history = newHist
}

// TODO: Color repository
func (strip *TabStrip) Draw(ctx *Context) {
	x := 0
	for i, tab := range strip.Tabs {
		style := strip.uiConfig.GetStyle(config.STYLE_TAB)
		if strip.Selected == i {
			style = strip.uiConfig.GetStyleSelected(config.STYLE_TAB)
		}
		tabWidth := 32
		if ctx.Width()-x < tabWidth {
			tabWidth = ctx.Width() - x - 2
		}
		name := tab.Name
		if tab.pinned {
			name = strip.uiConfig.PinnedTabMarker + name
		}
		trunc := runewidth.Truncate(name, tabWidth, "…")
		x += ctx.Printf(x, 0, style, " %s ", trunc)
		if x >= ctx.Width() {
			break
		}
	}
	ctx.Fill(x, 0, ctx.Width()-x, 1, ' ',
		strip.uiConfig.GetStyle(config.STYLE_TAB))
}

func (strip *TabStrip) Invalidate() {
	if strip.onInvalidateStrip != nil {
		strip.onInvalidateStrip(strip)
	}
}

func (strip *TabStrip) MouseEvent(localX int, localY int, event tcell.Event) {
	changeFocus := func(focus bool) {
		interactive, ok := strip.parent.Tabs[strip.parent.Selected].Content.(Interactive)
		if ok {
			interactive.Focus(focus)
		}
	}
	unfocus := func() { changeFocus(false) }
	refocus := func() { changeFocus(true) }
	switch event := event.(type) {
	case *tcell.EventMouse:
		switch event.Buttons() {
		case tcell.Button1:
			selectedTab, ok := strip.Clicked(localX, localY)
			if !ok || selectedTab == strip.parent.Selected {
				return
			}
			unfocus()
			strip.parent.Select(selectedTab)
			refocus()
		case tcell.WheelDown:
			unfocus()
			strip.parent.NextTab()
			refocus()
		case tcell.WheelUp:
			unfocus()
			strip.parent.PrevTab()
			refocus()
		case tcell.Button3:
			selectedTab, ok := strip.Clicked(localX, localY)
			if !ok {
				return
			}
			unfocus()
			if selectedTab == strip.parent.Selected {
				strip.parent.CloseTab(selectedTab)
			} else {
				current := strip.parent.Selected
				strip.parent.CloseTab(selectedTab)
				strip.parent.Select(current)
			}
			refocus()
		}
	}
}

func (strip *TabStrip) OnInvalidate(onInvalidate func(d Drawable)) {
	strip.onInvalidateStrip = onInvalidate
}

func (strip *TabStrip) Clicked(mouseX int, mouseY int) (int, bool) {
	x := 0
	for i, tab := range strip.Tabs {
		trunc := runewidth.Truncate(tab.Name, 32, "…")
		length := len(trunc) + 2
		if x <= mouseX && mouseX < x+length {
			return i, true
		}
		x += length
	}
	return 0, false
}

func (content *TabContent) Children() []Drawable {
	children := make([]Drawable, len(content.Tabs))
	for i, tab := range content.Tabs {
		children[i] = tab.Content
	}
	return children
}

func (content *TabContent) Draw(ctx *Context) {
	if content.Selected >= len(content.Tabs) {
		width := ctx.Width()
		height := ctx.Height()
		ctx.Fill(0, 0, width, height, ' ',
			content.uiConfig.GetStyle(config.STYLE_TAB))
	}

	tab := content.Tabs[content.Selected]
	tab.Content.Draw(ctx)
}

func (content *TabContent) MouseEvent(localX int, localY int, event tcell.Event) {
	tab := content.Tabs[content.Selected]
	switch tabContent := tab.Content.(type) {
	case Mouseable:
		tabContent.MouseEvent(localX, localY, event)
	}
}

func (content *TabContent) Invalidate() {
	if content.onInvalidateContent != nil {
		content.onInvalidateContent(content)
	}
	tab := content.Tabs[content.Selected]
	tab.Content.Invalidate()
}

func (content *TabContent) OnInvalidate(onInvalidate func(d Drawable)) {
	content.onInvalidateContent = onInvalidate
}
