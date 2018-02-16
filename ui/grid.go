package ui

import "fmt"

type Grid struct {
	Rows         []DimSpec
	Columns      []DimSpec
	Cells        []*GridCell
	onInvalidate func(d Drawable)
}

const (
	SIZE_EXACT  = iota
	SIZE_WEIGHT = iota
)

// Specifies the layout of a single row or column
type DimSpec struct {
	// One of SIZE_EXACT or SIZE_WEIGHT
	Strategy uint
	// If Strategy = SIZE_EXACT, this is the number of cells this dim shall
	// occupy. If SIZE_WEIGHT, the space left after all exact dims are measured
	// is distributed amonst the remaining dims weighted by this value.
	Size *uint
}

type GridCell struct {
	Row     uint
	Column  uint
	RowSpan uint
	ColSpan uint
	Content Drawable
	invalid bool
}

func (grid *Grid) Draw(ctx Context) {
	// TODO
}

func (grid *Grid) OnInvalidate(onInvalidate func(d Drawable)) {
	grid.onInvalidate = onInvalidate
}

func (grid *Grid) AddChild(cell *GridCell) {
	grid.Cells = append(grid.Cells, cell)
	cell.Content.OnInvalidate(grid.cellInvalidated)
	cell.invalid = true
}

func (grid *Grid) RemoveChild(cell *GridCell) {
	for i, _cell := range grid.Cells {
		if _cell == cell {
			grid.Cells = append(grid.Cells[:i], grid.Cells[i+1:]...)
			break
		}
	}
}

func (grid *Grid) cellInvalidated(drawable Drawable) {
	var cell *GridCell
	for _, cell = range grid.Cells {
		if cell.Content == drawable {
			break
		}
		cell = nil
	}
	if cell == nil {
		panic(fmt.Errorf("Attempted to invalidate unknown cell"))
	}
	cell.invalid = true
	if grid.onInvalidate != nil {
		grid.onInvalidate(grid)
	}
}
