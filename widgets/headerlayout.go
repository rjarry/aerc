package widgets

import (
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/models"
)

type HeaderLayout [][]string

// forMessage returns a filtered header layout, removing rows whose headers
// do not appear in the provided message.
func (layout HeaderLayout) forMessage(msg *models.MessageInfo) HeaderLayout {
	headers := msg.RFC822Headers
	result := make(HeaderLayout, 0, len(layout))
	for _, row := range layout {
		// To preserve layout alignment, only hide rows if all columns are empty
		for _, col := range row {
			if headers.Get(col) != "" {
				result = append(result, row)
				break
			}
		}
	}
	return result
}

// grid builds a ui grid, populating each cell by calling a callback function
// with the current header string.
func (layout HeaderLayout) grid(cb func(string) ui.Drawable) (grid *ui.Grid, height int) {
	rowCount := len(layout) + 1 // extra row for spacer
	grid = ui.MakeGrid(rowCount, 1, ui.SIZE_EXACT, ui.SIZE_WEIGHT)
	for i, cols := range layout {
		r := ui.MakeGrid(1, len(cols), ui.SIZE_EXACT, ui.SIZE_WEIGHT)
		for j, col := range cols {
			r.AddChild(cb(col)).At(0, j)
		}
		grid.AddChild(r).At(i, 0)
	}
	grid.AddChild(ui.NewFill(' ')).At(rowCount-1, 0)
	return grid, rowCount
}
