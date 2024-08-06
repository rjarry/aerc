package app

import (
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Dialog interface {
	ui.DrawableInteractive
	ContextWidth() (func(int) int, func(int) int)
	ContextHeight() (func(int) int, func(int) int)
}

type dialog struct {
	ui.DrawableInteractive
	x func(int) int
	y func(int) int
	w func(int) int
	h func(int) int
}

func (d *dialog) ContextWidth() (func(int) int, func(int) int) {
	return d.x, d.w
}

func (d *dialog) ContextHeight() (func(int) int, func(int) int) {
	return d.y, d.h
}

func NewDialog(
	d ui.DrawableInteractive,
	x func(int) int, y func(int) int,
	w func(int) int, h func(int) int,
) *dialog {
	return &dialog{DrawableInteractive: d, x: x, y: y, w: w, h: h}
}

// DefaultDialog creates a dialog window spanning half of the screen
func DefaultDialog(d ui.DrawableInteractive) Dialog {
	position := SelectedAccountUiConfig().DialogPosition
	width := SelectedAccountUiConfig().DialogWidth
	height := SelectedAccountUiConfig().DialogHeight
	return NewDialog(d,
		// horizontal starting position in columns from the left
		func(w int) int {
			return (w * (100 - width)) / 200
		},
		// vertical starting position in lines from the top
		func(h int) int {
			switch position {
			case "center":
				return (h * (100 - height)) / 200
			case "bottom":
				return h - (h * height / 100)
			default:
				return 1
			}
		},
		// dialog width from the starting column
		func(w int) int {
			return w * width / 100
		},
		// dialog height from the starting line
		func(h int) int {
			if position == "bottom" {
				return h*height/100 - 1
			}
			return h * height / 100
		},
	)
}
