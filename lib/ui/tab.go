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

func (tabs *Tabs) Add(content Drawable, name string) {
	tabs.Tabs = append(tabs.Tabs, &Tab{
		Content: content,
		Name:    name,
	})
	tabs.TabStrip.Invalidate()
	content.OnInvalidate(tabs.invalidateChild)
}

func (tabs *Tabs) invalidateChild(d Drawable) {
	for i, tab := range tabs.Tabs {
		if tab.Content == d {
			if i == tabs.Selected {
				tabs.TabContent.Invalidate()
			}
			return
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
	tabs.TabStrip.Invalidate()
}

func (tabs *Tabs) Select(index int) {
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
		style := tcell.StyleDefault.
			Background(tcell.ColorWhite).
			Foreground(tcell.ColorBlack)
		if strip.Selected == i {
			style = tcell.StyleDefault.
				Background(tcell.ColorDefault).
				Foreground(tcell.ColorDefault)
		}
		x += ctx.Printf(x, 0, style, " %s ", tab.Name)
	}
	style := tcell.StyleDefault.
		Background(tcell.ColorWhite).
		Foreground(tcell.ColorBlack)
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

func (content *TabContent) Draw(ctx *Context) {
	tab := content.Tabs[content.Selected]
	tab.Content.Draw(ctx)
}

func (content *TabContent) Invalidate() {
	if content.onInvalidateContent != nil {
		content.onInvalidateContent(content)
	}
}

func (content *TabContent) OnInvalidate(onInvalidate func(d Drawable)) {
	content.onInvalidateContent = onInvalidate
}
