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
