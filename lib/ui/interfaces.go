package ui

import (
	"github.com/gdamore/tcell/v2"
)

// Drawable is a UI component that can draw. Unless specified, all methods must
// only be called from a single goroutine, the UI goroutine.
type Drawable interface {
	// Called when this renderable should draw itself.
	Draw(ctx *Context)
	// Specifies a function to call when this cell needs to be redrawn. The
	// callback may be called in any goroutine.
	OnInvalidate(callback func(d Drawable))
	// Invalidates the drawable. This can be called from any goroutine.
	Invalidate()
}

type RootDrawable interface {
	Initialize(ui *UI)
}

type Interactive interface {
	// Returns true if the event was handled by this component
	Event(event tcell.Event) bool
	// Indicates whether or not this control will receive input events
	Focus(focus bool)
}

type Beeper interface {
	OnBeep(func() error)
}

type DrawableInteractive interface {
	Drawable
	Interactive
}

type DrawableInteractiveBeeper interface {
	DrawableInteractive
	Beeper
}

// A drawable which contains other drawables
type Container interface {
	Drawable
	// Return all of the drawables which are children of this one (do not
	// recurse into your grandchildren).
	Children() []Drawable
}

type MouseHandler interface {
	// Handle a mouse event which occurred at the local x and y positions
	MouseEvent(localX int, localY int, event tcell.Event)
}

// A drawable that can be interacted with by the mouse
type Mouseable interface {
	Drawable
	MouseHandler
}

type MouseableDrawableInteractive interface {
	DrawableInteractive
	MouseHandler
}
