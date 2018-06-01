package ui

import (
	"github.com/gdamore/tcell"
)

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
