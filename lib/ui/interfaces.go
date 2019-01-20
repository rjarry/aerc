package ui

import (
	"github.com/gdamore/tcell"
)

type Drawable interface {
	// Called when this renderable should draw itself
	Draw(ctx *Context)
	// Specifies a function to call when this cell needs to be redrawn
	OnInvalidate(callback func(d Drawable))
	// Invalidates the drawable
	Invalidate()
}

type Interactive interface {
	// Returns true if the event was handled by this component
	Event(event tcell.Event) bool
}

type Simulator interface {
	// Queues up the given input events for simulation
	Simulate(events []tcell.Event)
}

type DrawableInteractive interface {
	Drawable
	Interactive
}

// A drawable which contains other drawables
type Container interface {
	Drawable
	// A list of all drawables which are children of this one (do not recurse
	// into your grandchildren).
	Children() []Drawable
	// Return the "focused" child, or none of no preference. Does not actually
	// have to be Interactive. If there is a preferred child, input events will
	// be directed to it. If there's no preference, events will be delivered to
	// all children.
	InteractiveChild() Drawable
}
