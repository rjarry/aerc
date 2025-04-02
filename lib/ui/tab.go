package ui

import (
	"sync"

	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rockorager/vaxis"
)

const tabRuneWidth int = 32 // TODO: make configurable

type Tabs struct {
	tabs       []*Tab
	TabStrip   *TabStrip
	TabContent *TabContent
	curIndex   int
	history    []*Tab
	m          sync.Mutex

	ui func(d Drawable) *config.UIConfig

	parent   *Tabs //nolint:structcheck // used within this file
	CloseTab func(index int)
}

type Tab struct {
	Content        Drawable
	Name           string
	pinned         bool
	indexBeforePin int
	title          string
}

func (t *Tab) SetTitle(s string) {
	t.title = s
}

func (t *Tab) displayName(pinMarker string) string {
	name := t.Name
	if t.title != "" {
		name = t.title
	}
	if t.pinned {
		name = pinMarker + name
	}
	return name
}

type (
	TabStrip   Tabs
	TabContent Tabs
)

func NewTabs(ui func(d Drawable) *config.UIConfig) *Tabs {
	tabs := &Tabs{ui: ui}
	tabs.TabStrip = (*TabStrip)(tabs)
	tabs.TabStrip.parent = tabs
	tabs.TabContent = (*TabContent)(tabs)
	tabs.TabContent.parent = tabs
	return tabs
}

func (tabs *Tabs) Add(content Drawable, name string, background bool) *Tab {
	tab := &Tab{
		Content: content,
		Name:    name,
	}
	tabs.tabs = append(tabs.tabs, tab)
	if !background {
		tabs.selectPriv(len(tabs.tabs)-1, true)
	}
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

func (tabs *Tabs) Remove(content Drawable) {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	index := -1
	for i, tab := range tabs.tabs {
		if tab.Content == content {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}

	tab := tabs.tabs[index]
	if vis, ok := tab.Content.(Visible); ok {
		vis.Show(false)
	}
	if vis, ok := tab.Content.(Focusable); ok {
		vis.Focus(false)
	}
	tabs.tabs = append(tabs.tabs[:index], tabs.tabs[index+1:]...)
	tabs.removeHistory(tab)

	if index == tabs.curIndex {
		// only pop the tab history if the closing tab is selected
		prevIndex, ok := tabs.popHistory()
		if !ok {
			if tabs.curIndex < len(tabs.tabs) {
				// history is empty, select tab on the right if possible
				prevIndex = tabs.curIndex
			} else {
				// if removing the last tab, select the now last tab
				prevIndex = len(tabs.tabs) - 1
			}
		}
		tabs.selectPriv(prevIndex, false)
	} else if index < tabs.curIndex {
		// selected tab is now one to the left of where it was
		tabs.selectPriv(tabs.curIndex-1, false)
	}
	Invalidate()
}

func (tabs *Tabs) Replace(contentSrc Drawable, contentTarget Drawable, name string) {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	for i, tab := range tabs.tabs {
		if tab.Content == contentSrc {
			if vis, ok := tab.Content.(Visible); ok {
				vis.Show(false)
			}
			if vis, ok := tab.Content.(Focusable); ok {
				vis.Focus(false)
			}
			tab.Content = contentTarget
			tabs.selectPriv(i, false)
			Invalidate()
			break
		}
	}
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
	return tabs.selectPriv(index, true)
}

func (tabs *Tabs) selectPriv(index int, unselectPrev bool) bool {
	if index < 0 || index >= len(tabs.tabs) {
		return false
	}

	// only push valid tabs onto the history
	if unselectPrev && tabs.curIndex < len(tabs.tabs) {
		prev := tabs.tabs[tabs.curIndex]
		if vis, ok := prev.Content.(Visible); ok {
			vis.Show(false)
		}
		if vis, ok := prev.Content.(Focusable); ok {
			vis.Focus(false)
		}
		tabs.pushHistory(prev)
	}

	next := tabs.tabs[index]
	if vis, ok := next.Content.(Visible); ok {
		vis.Show(true)
	}
	if vis, ok := next.Content.(Focusable); ok {
		vis.Focus(true)
	}
	tabs.curIndex = index
	Invalidate()

	return true
}

func (tabs *Tabs) SelectName(name string) bool {
	tabs.m.Lock()
	defer tabs.m.Unlock()
	for i, tab := range tabs.tabs {
		if tab.Name == name {
			return tabs.selectPriv(i, true)
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
	return tabs.selectPriv(index, true)
}

func (tabs *Tabs) SelectOffset(offset int) {
	tabs.m.Lock()
	tabCount := len(tabs.tabs)
	newIndex := (tabs.curIndex + offset) % tabCount
	if newIndex < 0 {
		// Handle negative offsets correctly
		newIndex += tabCount
	}
	tabs.selectPriv(newIndex, true)
	tabs.m.Unlock()
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
	tabs.tabs[from], tabs.tabs[to] = tabs.tabs[to], tabs.tabs[from]
	tabs.curIndex = to
	Invalidate()
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
	tabs.selectPriv(next, true)
	tabs.m.Unlock()
}

func (tabs *Tabs) PrevTab() {
	tabs.m.Lock()
	next := tabs.curIndex - 1
	if next < 0 {
		next = len(tabs.tabs) - 1
	}
	tabs.selectPriv(next, true)
	tabs.m.Unlock()
}

const maxHistory = 256

func (tabs *Tabs) pushHistory(tab *Tab) {
	tabs.history = append(tabs.history, tab)
	if len(tabs.history) > maxHistory {
		tabs.history = tabs.history[1:]
	}
}

func (tabs *Tabs) popHistory() (int, bool) {
	if len(tabs.history) == 0 {
		return -1, false
	}
	tab := tabs.history[len(tabs.history)-1]
	tabs.history = tabs.history[:len(tabs.history)-1]
	index := -1
	for i, t := range tabs.tabs {
		if t == tab {
			index = i
			break
		}
	}
	if index == -1 {
		return -1, false
	}
	return index, true
}

func (tabs *Tabs) removeHistory(tab *Tab) {
	var newHist []*Tab
	for _, item := range tabs.history {
		if item != tab {
			newHist = append(newHist, item)
		}
	}
	tabs.history = newHist
}

// TODO: Color repository
func (strip *TabStrip) Draw(ctx *Context) {
	x := 0
	strip.parent.m.Lock()
	for i, tab := range strip.tabs {
		uiConfig := strip.ui(tab.Content)
		if uiConfig == nil {
			uiConfig = config.Ui
		}
		style := uiConfig.GetStyle(config.STYLE_TAB)
		if strip.curIndex == i {
			style = uiConfig.GetStyleSelected(config.STYLE_TAB)
		}
		tabWidth := tabRuneWidth
		if ctx.Width()-x < tabWidth {
			tabWidth = ctx.Width() - x - 2
		}
		name := tab.displayName(uiConfig.PinnedTabMarker)
		trunc := runewidth.Truncate(name, tabWidth, "…")
		x += ctx.Printf(x, 0, style, " %s ", trunc)
		if x >= ctx.Width() {
			break
		}
	}
	strip.parent.m.Unlock()
	ctx.Fill(x, 0, ctx.Width()-x, 1, ' ',
		config.Ui.GetStyle(config.STYLE_TAB))
}

func (strip *TabStrip) Invalidate() {
	Invalidate()
}

func (strip *TabStrip) MouseEvent(localX int, localY int, event vaxis.Event) {
	strip.parent.m.Lock()
	defer strip.parent.m.Unlock()
	changeFocus := func(focus bool) {
		focusable, ok := strip.parent.tabs[strip.parent.curIndex].Content.(Focusable)
		if ok {
			focusable.Focus(focus)
		}
	}
	unfocus := func() { changeFocus(false) }
	refocus := func() { changeFocus(true) }
	if event, ok := event.(vaxis.Mouse); ok {
		switch event.Button {
		case vaxis.MouseLeftButton:
			selectedTab, ok := strip.clicked(localX, localY)
			if !ok || selectedTab == strip.parent.curIndex {
				return
			}
			unfocus()
			strip.parent.selectPriv(selectedTab, true)
			refocus()
		case vaxis.MouseWheelDown:
			unfocus()
			index := strip.parent.curIndex + 1
			if index >= len(strip.parent.tabs) {
				index = 0
			}
			strip.parent.selectPriv(index, true)
			refocus()
		case vaxis.MouseWheelUp:
			unfocus()
			index := strip.parent.curIndex - 1
			if index < 0 {
				index = len(strip.parent.tabs) - 1
			}
			strip.parent.selectPriv(index, true)
			refocus()
		case vaxis.MouseMiddleButton:
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

func (strip *TabStrip) clicked(mouseX int, mouseY int) (int, bool) {
	x := 0
	for i, tab := range strip.tabs {
		uiConfig := strip.ui(tab.Content)
		if uiConfig == nil {
			uiConfig = config.Ui
		}
		name := tab.displayName(uiConfig.PinnedTabMarker)
		trunc := runewidth.Truncate(name, tabRuneWidth, "…")
		length := runewidth.StringWidth(trunc) + 2
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
			config.Ui.GetStyle(config.STYLE_TAB))
	}
	tab := content.tabs[content.curIndex]
	content.parent.m.Unlock()
	tab.Content.Draw(ctx)
}

func (content *TabContent) MouseEvent(localX int, localY int, event vaxis.Event) {
	content.parent.m.Lock()
	tab := content.tabs[content.curIndex]
	content.parent.m.Unlock()
	if tabContent, ok := tab.Content.(Mouseable); ok {
		tabContent.MouseEvent(localX, localY, event)
	}
}

func (content *TabContent) Invalidate() {
	Invalidate()
}
