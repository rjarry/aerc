package ui

import (
	"git.sr.ht/~rockorager/vaxis"
)

// Drawable is a UI component that can draw. Unless specified, all methods must
// only be called from a single goroutine, the UI goroutine.
type Drawable interface {
	// Called when this renderable should draw itself.
	Draw(ctx *Context)
	// Invalidates the UI. This can be called from any goroutine.
	Invalidate()
}

type Closeable interface {
	Close()
}

type Visible interface {
	// Indicate that this component is visible or not
	Show(bool)
}

type Focusable interface {
	// Indicates whether or not this control will receive input events
	Focus(focus bool)
}

type Interactive interface {
	Focusable
	// Returns true if the event was handled by this component
	Event(event vaxis.Event) bool
}

type Beeper interface {
	OnBeep(func())
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
	MouseEvent(localX int, localY int, event vaxis.Event)
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
