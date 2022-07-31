package ui

import (
	"io"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
)

type Tabs struct {
	tabs       []*Tab
	TabStrip   *TabStrip
	TabContent *TabContent
	curIndex   int
	history    []int
	m          sync.Mutex

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
	uiConf         *config.UIConfig
}

type (
	TabStrip   Tabs
	TabContent Tabs
)

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

func (tabs *Tabs) Add(content Drawable, name string, uiConf *config.UIConfig) *Tab {
	tab := &Tab{
		Content: content,
		Name:    name,
		uiConf:  uiConf,
	}
	tabs.tabs = append(tabs.tabs, tab)
	tabs.selectPriv(len(tabs.tabs) - 1)
	content.OnInvalidate(tabs.invalidateChild)
	return tab
}

func (tabs *Tabs) Names() []string {
	var names []string
	tabs.m.Lock()
	for _, tab := range tabs.tabs {
		names = append(names, tab.Name)
	}
	tabs.m.Unlock()
	return names
}

func (tabs *Tabs) invalidateChild(d Drawable) {
	if tabs.curIndex >= len(tabs.tabs) {
		return
	}

	if tabs.tabs[tabs.curIndex].Content == d {
		if tabs.onInvalidateContent != nil {
			tabs.onInvalidateContent(tabs.TabContent)
		}
	}
}

func (tabs *Tabs) Remove(content Drawable) {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	indexToRemove := -1
	for i, tab := range tabs.tabs {
		if tab.Content == content {
			tabs.tabs = append(tabs.tabs[:i], tabs.tabs[i+1:]...)
			tabs.removeHistory(i)
			indexToRemove = i
			break
		}
	}
	if indexToRemove < 0 {
		return
	}
	// only pop the tab history if the closing tab is selected
	if indexToRemove == tabs.curIndex {
		index, ok := tabs.popHistory()
		if ok {
			tabs.selectPriv(index)
		}
	} else if indexToRemove < tabs.curIndex {
		// selected tab is now one to the left of where it was
		tabs.selectPriv(tabs.curIndex - 1)
	}
	interactive, ok := tabs.tabs[tabs.curIndex].Content.(Interactive)
	if ok {
		interactive.Focus(true)
	}
}

func (tabs *Tabs) Replace(contentSrc Drawable, contentTarget Drawable, name string) {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	replaceTab := &Tab{
		Content: contentTarget,
		Name:    name,
	}
	for i, tab := range tabs.tabs {
		if tab.Content == contentSrc {
			tabs.tabs[i] = replaceTab
			tabs.selectPriv(i)
			if c, ok := contentSrc.(io.Closer); ok {
				c.Close()
			}
			break
		}
	}
	tabs.TabStrip.Invalidate()
	contentTarget.OnInvalidate(tabs.invalidateChild)
}

func (tabs *Tabs) Get(index int) *Tab {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	if index < 0 || index >= len(tabs.tabs) {
		return nil
	}
	return tabs.tabs[index]
}

func (tabs *Tabs) Selected() *Tab {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	if tabs.curIndex < 0 || tabs.curIndex >= len(tabs.tabs) {
		return nil
	}
	return tabs.tabs[tabs.curIndex]
}

func (tabs *Tabs) Select(index int) bool {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	return tabs.selectPriv(index)
}

func (tabs *Tabs) selectPriv(index int) bool {
	if index < 0 || index >= len(tabs.tabs) {
		return false
	}

	if tabs.curIndex != index {
		// only push valid tabs onto the history
		if tabs.curIndex < len(tabs.tabs) {
			tabs.pushHistory(tabs.curIndex)
		}
		tabs.curIndex = index
		tabs.TabStrip.Invalidate()
		tabs.TabContent.Invalidate()
	}
	return true
}

func (tabs *Tabs) SelectName(name string) bool {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	for i, tab := range tabs.tabs {
		if tab.Name == name {
			return tabs.selectPriv(i)
		}
	}
	return false
}

func (tabs *Tabs) SelectPrevious() bool {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	index, ok := tabs.popHistory()
	if !ok {
		return false
	}
	return tabs.selectPriv(index)
}

func (tabs *Tabs) MoveTab(to int, relative bool) {
	tabs.m.Lock()
	tabs.moveTabPriv(to, relative)
	tabs.m.Unlock()
}

func (tabs *Tabs) moveTabPriv(to int, relative bool) {
	from := tabs.curIndex

	if relative {
		to = from + to
	}
	if to < 0 {
		to = 0
	}
	if to >= len(tabs.tabs) {
		to = len(tabs.tabs) - 1
	}

	tab := tabs.tabs[from]
	if to > from {
		copy(tabs.tabs[from:to], tabs.tabs[from+1:to+1])
		for i, h := range tabs.history {
			if h == from {
				tabs.history[i] = to
			}
			if h > from && h <= to {
				tabs.history[i] -= 1
			}
		}
	} else if from > to {
		copy(tabs.tabs[to+1:from+1], tabs.tabs[to:from])
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

	tabs.tabs[to] = tab
	tabs.curIndex = to
	tabs.TabStrip.Invalidate()
}

func (tabs *Tabs) PinTab() {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	if tabs.tabs[tabs.curIndex].pinned {
		return
	}

	pinEnd := len(tabs.tabs)
	for i, t := range tabs.tabs {
		if !t.pinned {
			pinEnd = i
			break
		}
	}

	for _, t := range tabs.tabs {
		if t.pinned && t.indexBeforePin > tabs.curIndex-pinEnd {
			t.indexBeforePin -= 1
		}
	}

	tabs.tabs[tabs.curIndex].pinned = true
	tabs.tabs[tabs.curIndex].indexBeforePin = tabs.curIndex - pinEnd

	tabs.moveTabPriv(pinEnd, false)
}

func (tabs *Tabs) UnpinTab() {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	if !tabs.tabs[tabs.curIndex].pinned {
		return
	}

	pinEnd := len(tabs.tabs)
	for i, t := range tabs.tabs {
		if i != tabs.curIndex && t.pinned && t.indexBeforePin > tabs.tabs[tabs.curIndex].indexBeforePin {
			t.indexBeforePin += 1
		}
		if !t.pinned {
			pinEnd = i
			break
		}
	}

	tabs.tabs[tabs.curIndex].pinned = false

	tabs.moveTabPriv(tabs.tabs[tabs.curIndex].indexBeforePin+pinEnd-1, false)
}

func (tabs *Tabs) NextTab() {
	tabs.m.Lock()
	next := tabs.curIndex + 1
	if next >= len(tabs.tabs) {
		next = 0
	}
	tabs.selectPriv(next)
	tabs.m.Unlock()
}

func (tabs *Tabs) PrevTab() {
	tabs.m.Lock()
	next := tabs.curIndex - 1
	if next < 0 {
		next = len(tabs.tabs) - 1
	}
	tabs.selectPriv(next)
	tabs.m.Unlock()
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
	strip.parent.m.Lock()
	for i, tab := range strip.tabs {
		uiConfig := strip.uiConfig
		if tab.uiConf != nil {
			uiConfig = tab.uiConf
		}
		style := uiConfig.GetStyle(config.STYLE_TAB)
		if strip.curIndex == i {
			style = uiConfig.GetStyleSelected(config.STYLE_TAB)
		}
		tabWidth := 32
		if ctx.Width()-x < tabWidth {
			tabWidth = ctx.Width() - x - 2
		}
		name := tab.Name
		if tab.pinned {
			name = uiConfig.PinnedTabMarker + name
		}
		trunc := runewidth.Truncate(name, tabWidth, "…")
		x += ctx.Printf(x, 0, style, " %s ", trunc)
		if x >= ctx.Width() {
			break
		}
	}
	strip.parent.m.Unlock()
	ctx.Fill(x, 0, ctx.Width()-x, 1, ' ',
		strip.uiConfig.GetStyle(config.STYLE_TAB))
}

func (strip *TabStrip) Invalidate() {
	if strip.onInvalidateStrip != nil {
		strip.onInvalidateStrip(strip)
	}
}

func (strip *TabStrip) MouseEvent(localX int, localY int, event tcell.Event) {
	strip.parent.m.Lock()
	defer strip.parent.m.Unlock()
	changeFocus := func(focus bool) {
		interactive, ok := strip.parent.tabs[strip.parent.curIndex].Content.(Interactive)
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
			selectedTab, ok := strip.clicked(localX, localY)
			if !ok || selectedTab == strip.parent.curIndex {
				return
			}
			unfocus()
			strip.parent.selectPriv(selectedTab)
			refocus()
		case tcell.WheelDown:
			unfocus()
			index := strip.parent.curIndex + 1
			if index >= len(strip.parent.tabs) {
				index = 0
			}
			strip.parent.selectPriv(index)
			refocus()
		case tcell.WheelUp:
			unfocus()
			index := strip.parent.curIndex - 1
			if index < 0 {
				index = len(strip.parent.tabs) - 1
			}
			strip.parent.selectPriv(index)
			refocus()
		case tcell.Button3:
			selectedTab, ok := strip.clicked(localX, localY)
			if !ok {
				return
			}
			unfocus()
			strip.parent.m.Unlock()
			strip.parent.CloseTab(selectedTab)
			strip.parent.m.Lock()
			refocus()
		}
	}
}

func (strip *TabStrip) OnInvalidate(onInvalidate func(d Drawable)) {
	strip.onInvalidateStrip = onInvalidate
}

func (strip *TabStrip) clicked(mouseX int, mouseY int) (int, bool) {
	x := 0
	for i, tab := range strip.tabs {
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
	content.parent.m.Lock()
	children := make([]Drawable, len(content.tabs))
	for i, tab := range content.tabs {
		children[i] = tab.Content
	}
	content.parent.m.Unlock()
	return children
}

func (content *TabContent) Draw(ctx *Context) {
	content.parent.m.Lock()
	if content.curIndex >= len(content.tabs) {
		width := ctx.Width()
		height := ctx.Height()
		ctx.Fill(0, 0, width, height, ' ',
			content.uiConfig.GetStyle(config.STYLE_TAB))
	}
	tab := content.tabs[content.curIndex]
	content.parent.m.Unlock()
	tab.Content.Draw(ctx)
}

func (content *TabContent) MouseEvent(localX int, localY int, event tcell.Event) {
	content.parent.m.Lock()
	tab := content.tabs[content.curIndex]
	content.parent.m.Unlock()
	switch tabContent := tab.Content.(type) {
	case Mouseable:
		tabContent.MouseEvent(localX, localY, event)
	}
}

func (content *TabContent) Invalidate() {
	if content.onInvalidateContent != nil {
		content.onInvalidateContent(content)
	}
	tab := content.tabs[content.curIndex]
	tab.Content.Invalidate()
}

func (content *TabContent) OnInvalidate(onInvalidate func(d Drawable)) {
	content.onInvalidateContent = onInvalidate
}
