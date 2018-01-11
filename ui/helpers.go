package ui

import (
	"fmt"

	tb "github.com/nsf/termbox-go"
)

func TPrintf(geo *Geometry, ref tb.Cell, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	_geo := *geo
	newline := func() {
		// TODO: Abort when out of room?
		geo.Col = _geo.Col
		geo.Row++
	}
	for _, ch := range str {
		switch ch {
		case '\n':
			newline()
		case '\r':
			geo.Col = _geo.Col
		default:
			tb.SetCell(geo.Col, geo.Row, ch, ref.Fg, ref.Bg)
			geo.Col++
			if geo.Col == _geo.Col+geo.Width {
				newline()
			}
		}
	}
}

func TFill(geo Geometry, ref tb.Cell) {
	_geo := geo
	for ; geo.Row < geo.Height; geo.Row++ {
		for ; geo.Col < geo.Width; geo.Col++ {
			tb.SetCell(geo.Col, geo.Row, ref.Ch, ref.Fg, ref.Bg)
		}
		geo.Col = _geo.Col
	}
}
