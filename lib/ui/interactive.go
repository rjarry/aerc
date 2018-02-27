package ui

import (
	tb "github.com/nsf/termbox-go"
)

type Interactive interface {
	// Returns true if the event was handled by this component
	Event(event tb.Event) bool
}

type Simulator interface {
	// Queues up the given input events for simulation
	Simulate(events []tb.Event)
}
