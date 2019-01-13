package ui

import (
	"fmt"
)

// A container which arranges its children in a list.
type List struct {
	Items        []*ListItem
	itemHeight   int
	onInvalidate func(d Drawable)
	selected     int
}

type ListItem struct {
	Content Drawable
	invalid bool
}

type SelectableDrawable interface {
	Drawable
	DrawWithSelected(ctx *Context, selected bool)
}

func NewList() *List {
	return &List{itemHeight: 1, selected: -1}
}

func (list *List) OnInvalidate(onInvalidate func(d Drawable)) {
	list.onInvalidate = onInvalidate
}

func (list *List) Invalidate() {
	for _, item := range list.Items {
		item.Content.Invalidate()
	}
	if list.onInvalidate != nil {
		list.onInvalidate(list)
	}
}

func (list *List) Draw(ctx *Context) {
	for i, item := range list.Items {
		if !item.invalid {
			continue
		}
		subctx := ctx.Subcontext(0, i, ctx.Width(), list.itemHeight)
		if content, ok := item.Content.(SelectableDrawable); ok {
			content.DrawWithSelected(subctx, i == list.selected)
		} else {
			item.Content.Draw(subctx)
		}
	}
}

func (list *List) Add(child Drawable) {
	list.Items = append(list.Items, &ListItem{Content: child, invalid: true})
	child.OnInvalidate(list.childInvalidated)
	list.Invalidate()
}

func (list *List) Remove(child Drawable) {
	for i, item := range list.Items {
		if item.Content == child {
			list.Items = append(list.Items[:i], list.Items[i+1:]...)
			child.OnInvalidate(nil)
			list.Invalidate()
			return
		}
	}
	panic(fmt.Errorf("Attempted to remove unknown child"))
}

func (list *List) Set(items []Drawable) {
	for _, item := range list.Items {
		item.Content.OnInvalidate(nil)
	}
	list.Items = make([]*ListItem, len(items))
	for i, item := range items {
		list.Items[i] = &ListItem{Content: item, invalid: true}
		item.OnInvalidate(list.childInvalidated)
	}
	list.Invalidate()
}

func (list *List) Select(index int) {
	if index >= len(list.Items) || index < 0 {
		panic(fmt.Errorf("Attempted to select unknown child"))
	}
	list.selected = index
	list.Invalidate()
}

func (list *List) ItemHeight(height int) {
	list.itemHeight = height
	list.Invalidate()
}

func (list *List) childInvalidated(child Drawable) {
	for _, item := range list.Items {
		if item.Content == child {
			item.invalid = true
			if list.onInvalidate != nil {
				list.onInvalidate(list)
			}
			return
		}
	}
	panic(fmt.Errorf("Attempted to invalidate unknown child"))
}
