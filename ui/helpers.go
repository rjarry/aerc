package ui

import (
	"fmt"

	tb "github.com/nsf/termbox-go"
)

func TPrintf(geo *Geometry, ref tb.Cell, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	_geo := *geo
	for _, ch := range str {
		tb.SetCell(geo.Col, geo.Row, ch, ref.Fg, ref.Bg)
		geo.Col++
		if geo.Col == _geo.Col+geo.Width {
			// TODO: Abort when out of room?
			geo.Col = _geo.Col
			geo.Row++
		}
	}
}
