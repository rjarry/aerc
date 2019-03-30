package ui

import (
	"github.com/gdamore/tcell"
)

type Tabs struct {
	Tabs       []*Tab
	TabStrip   *TabStrip
	TabContent *TabContent
	Selected   int

	onInvalidateStrip   func(d Drawable)
	onInvalidateContent func(d Drawable)
}

type Tab struct {
	Content Drawable
	Name    string
	invalid bool
}

type TabStrip Tabs
type TabContent Tabs

func NewTabs() *Tabs {
	tabs := &Tabs{}
	tabs.TabStrip = (*TabStrip)(tabs)
	tabs.TabContent = (*TabContent)(tabs)
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
	for i, tab := range tabs.Tabs {
		if tab.Content == content {
			tabs.Tabs = append(tabs.Tabs[:i], tabs.Tabs[i+1:]...)
			break
		}
	}
	/* Force the selected index into the existing range */
	if tabs.Selected >= len(tabs.Tabs) {
		tabs.Select(tabs.Selected - 1)
	}
	tabs.TabStrip.Invalidate()
}

func (tabs *Tabs) Select(index int) {
	if index >= len(tabs.Tabs) {
		panic("Tried to set tab index to a non-existing element")
	}

	if tabs.Selected != index {
		tabs.Selected = index
		tabs.TabStrip.Invalidate()
		tabs.TabContent.Invalidate()
	}
}

// TODO: Color repository
func (strip *TabStrip) Draw(ctx *Context) {
	x := 0
	for i, tab := range strip.Tabs {
		style := tcell.StyleDefault.Reverse(true)
		if strip.Selected == i {
			style = tcell.StyleDefault
		}
		x += ctx.Printf(x, 0, style, " %s ", tab.Name)
	}
	style := tcell.StyleDefault.Reverse(true)
	ctx.Fill(x, 0, ctx.Width()-x, 1, ' ', style)
}

func (strip *TabStrip) Invalidate() {
	if strip.onInvalidateStrip != nil {
		strip.onInvalidateStrip(strip)
	}
}

func (strip *TabStrip) OnInvalidate(onInvalidate func(d Drawable)) {
	strip.onInvalidateStrip = onInvalidate
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
		ctx.Fill(0, 0, width, height, ' ', tcell.StyleDefault)
	}

	tab := content.Tabs[content.Selected]
	tab.Content.Draw(ctx)
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
