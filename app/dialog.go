package app

import (
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Dialog interface {
	ui.DrawableInteractive
	ContextHeight() (func(int) int, func(int) int)
}

type dialog struct {
	ui.DrawableInteractive
	y func(int) int
	h func(int) int
}

func (d *dialog) ContextHeight() (func(int) int, func(int) int) {
	return d.y, d.h
}

func NewDialog(d ui.DrawableInteractive, y func(int) int, h func(int) int) Dialog {
	return &dialog{DrawableInteractive: d, y: y, h: h}
}

// DefaultDialog creates a dialog window spanning half of the screen
func DefaultDialog(d ui.DrawableInteractive) Dialog {
	return NewDialog(d,
		// vertical starting position in lines from the top
		func(h int) int {
			return h / 4
		},
		// dialog height from the starting line
		func(h int) int {
			return h / 2
		},
	)
}

// LargeDialog creates a dialog window spanning three quarter of the screen
func LargeDialog(d ui.DrawableInteractive) Dialog {
	return NewDialog(d,
		// vertical starting position in lines from the top
		func(h int) int {
			return h / 8
		},
		// dialog height from the starting line
		func(h int) int {
			return 3 * h / 4
		},
	)
}
