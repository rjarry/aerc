package ui

import (
	"git.sr.ht/~rockorager/vaxis"
)

func StyledString(s string) *vaxis.StyledString {
	return state.vx.NewStyledString(s, vaxis.Style{})
}

// Applies a style to a string. Any currently applied styles will not be overwritten
func ApplyStyle(style vaxis.Style, str string) string {
	ss := StyledString(str)
	d := vaxis.Style{}
	for i, sr := range ss.Cells {
		if sr.Style == d {
			sr.Style = style
			ss.Cells[i] = sr
		}
	}
	return ss.Encode()
}

// PadLeft inserts blank spaces at the beginning of the StyledString to produce
// a string of the provided width
func PadLeft(ss *vaxis.StyledString, width int) {
	w := ss.Len()
	if w >= width {
		return
	}
	cell := vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: " ",
			Width:    1,
		},
	}
	w = width - w
	cells := make([]vaxis.Cell, 0, len(ss.Cells)+w)
	for w > 0 {
		cells = append(cells, cell)
		w -= 1
	}
	cells = append(cells, ss.Cells...)
	ss.Cells = cells
}

// PadLeft inserts blank spaces at the end of the StyledString to produce
// a string of the provided width
func PadRight(ss *vaxis.StyledString, width int) {
	w := ss.Len()
	if w >= width {
		return
	}
	cell := vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: " ",
			Width:    1,
		},
	}
	w = width - w
	for w > 0 {
		w -= 1
		ss.Cells = append(ss.Cells, cell)
	}
}

// ApplyAttrs applies the style, and if another style is present ORs the
// attributes
func ApplyAttrs(ss *vaxis.StyledString, style vaxis.Style) {
	for i, cell := range ss.Cells {
		if style.Foreground != 0 {
			cell.Style.Foreground = style.Foreground
		}
		if style.Background != 0 {
			cell.Style.Background = style.Background
		}
		cell.Style.Attribute |= style.Attribute
		if style.UnderlineColor != 0 {
			cell.Style.UnderlineColor = style.UnderlineColor
		}
		if style.UnderlineStyle != vaxis.UnderlineOff {
			cell.Style.UnderlineStyle = style.UnderlineStyle
		}
		ss.Cells[i] = cell
	}
}

// Truncates the styled string on the right and inserts a '…' as the last
// character
func Truncate(ss *vaxis.StyledString, width int) {
	if ss.Len() <= width {
		return
	}
	cells := make([]vaxis.Cell, 0, len(ss.Cells))
	total := 0
	for _, cell := range ss.Cells {
		if total+cell.Width >= width {
			// we can't fit this cell so put in our truncator
			cells = append(cells, vaxis.Cell{
				Character: vaxis.Character{
					Grapheme: "…",
					Width:    1,
				},
				Style: cell.Style,
			})
			break
		}
		total += cell.Width
		cells = append(cells, cell)
	}
	ss.Cells = cells
}

// TruncateHead truncates the left side of the string and inserts '…' as the
// first character
func TruncateHead(ss *vaxis.StyledString, width int) {
	l := ss.Len()
	if l <= width {
		return
	}
	offset := l - width
	cells := make([]vaxis.Cell, 0, len(ss.Cells))
	cells = append(cells, vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: "…",
			Width:    1,
		},
	})
	total := 0
	for _, cell := range ss.Cells {
		total += cell.Width
		if total < offset {
			// we always have at least one for our truncator. We
			// copy this cells style to it so that it retains the
			// style information from the first printed cell
			cells[0].Style = cell.Style
			continue
		}
		cells = append(cells, cell)
	}
	ss.Cells = cells
}
