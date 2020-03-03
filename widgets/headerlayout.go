package widgets

import (
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/models"
)

type HeaderLayout [][]string

type HeaderLayoutFilter struct {
	layout HeaderLayout
	keep   func(msg *models.MessageInfo, header string) bool // filter criteria
}

// forMessage returns a filtered header layout, removing rows whose headers
// do not appear in the provided message.
func (filter HeaderLayoutFilter) forMessage(msg *models.MessageInfo) HeaderLayout {
	result := make(HeaderLayout, 0, len(filter.layout))
	for _, row := range filter.layout {
		// To preserve layout alignment, only hide rows if all columns are empty
		for _, col := range row {
			if filter.keep(msg, col) {
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
	rowCount := len(layout)
	grid = ui.MakeGrid(rowCount, 1, ui.SIZE_EXACT, ui.SIZE_WEIGHT)
	for i, cols := range layout {
		r := ui.MakeGrid(1, len(cols), ui.SIZE_EXACT, ui.SIZE_WEIGHT)
		for j, col := range cols {
			r.AddChild(cb(col)).At(0, j)
		}
		grid.AddChild(r).At(i, 0)
	}
	return grid, rowCount
}
