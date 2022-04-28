package ui

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
)

type Grid struct {
	Invalidatable
	rows         []GridSpec
	rowLayout    []gridLayout
	columns      []GridSpec
	columnLayout []gridLayout
	invalid      bool

	// Protected by mutex
	cells []*GridCell
	mutex sync.RWMutex
}

const (
	SIZE_EXACT  = iota
	SIZE_WEIGHT = iota
)

// Specifies the layout of a single row or column
type GridSpec struct {
	// One of SIZE_EXACT or SIZE_WEIGHT
	Strategy int

	// If Strategy = SIZE_EXACT, this function returns the number of cells
	// this row/col shall occupy. If SIZE_WEIGHT, the space left after all
	// exact rows/cols are measured is distributed amonst the remainder
	// weighted by the value returned by this function.
	Size func() int
}

// Used to cache layout of each row/column
type gridLayout struct {
	Offset int
	Size   int
}

type GridCell struct {
	Row     int
	Column  int
	RowSpan int
	ColSpan int
	Content Drawable
	invalid atomic.Value // bool
}

func NewGrid() *Grid {
	return &Grid{invalid: true}
}

// MakeGrid creates a grid with the specified number of columns and rows. Each
// cell has a size of 1.
func MakeGrid(numRows, numCols, rowStrategy, colStrategy int) *Grid {
	rows := make([]GridSpec, numRows)
	for i := 0; i < numRows; i++ {
		rows[i] = GridSpec{rowStrategy, Const(1)}
	}
	cols := make([]GridSpec, numCols)
	for i := 0; i < numCols; i++ {
		cols[i] = GridSpec{colStrategy, Const(1)}
	}
	return NewGrid().Rows(rows).Columns(cols)
}

func (cell *GridCell) At(row, col int) *GridCell {
	cell.Row = row
	cell.Column = col
	return cell
}

func (cell *GridCell) Span(rows, cols int) *GridCell {
	cell.RowSpan = rows
	cell.ColSpan = cols
	return cell
}

func (grid *Grid) Rows(spec []GridSpec) *Grid {
	grid.rows = spec
	return grid
}

func (grid *Grid) Columns(spec []GridSpec) *Grid {
	grid.columns = spec
	return grid
}

func (grid *Grid) Children() []Drawable {
	grid.mutex.RLock()
	defer grid.mutex.RUnlock()

	children := make([]Drawable, len(grid.cells))
	for i, cell := range grid.cells {
		children[i] = cell.Content
	}
	return children
}

func (grid *Grid) Draw(ctx *Context) {
	invalid := grid.invalid
	if invalid {
		grid.reflow(ctx)
	}

	grid.mutex.RLock()
	defer grid.mutex.RUnlock()

	for _, cell := range grid.cells {
		cellInvalid := cell.invalid.Load().(bool)
		if !cellInvalid && !invalid {
			continue
		}
		rows := grid.rowLayout[cell.Row : cell.Row+cell.RowSpan]
		cols := grid.columnLayout[cell.Column : cell.Column+cell.ColSpan]
		x := cols[0].Offset
		y := rows[0].Offset
		if x < 0 || y < 0 {
			continue
		}

		width := 0
		height := 0
		for _, col := range cols {
			width += col.Size
		}
		for _, row := range rows {
			height += row.Size
		}
		if x+width > ctx.Width() {
			width = ctx.Width() - x
		}
		if y+height > ctx.Height() {
			height = ctx.Height() - y
		}
		if width <= 0 || height <= 0 {
			continue
		}
		subctx := ctx.Subcontext(x, y, width, height)
		cell.Content.Draw(subctx)
	}
}

func (grid *Grid) MouseEvent(localX int, localY int, event tcell.Event) {
	switch event := event.(type) {
	case *tcell.EventMouse:
		invalid := grid.invalid

		grid.mutex.RLock()
		defer grid.mutex.RUnlock()

		for _, cell := range grid.cells {
			cellInvalid := cell.invalid.Load().(bool)
			if !cellInvalid && !invalid {
				continue
			}
			rows := grid.rowLayout[cell.Row : cell.Row+cell.RowSpan]
			cols := grid.columnLayout[cell.Column : cell.Column+cell.ColSpan]
			x := cols[0].Offset
			y := rows[0].Offset
			width := 0
			height := 0
			for _, col := range cols {
				width += col.Size
			}
			for _, row := range rows {
				height += row.Size
			}
			if x <= localX && localX < x+width && y <= localY && localY < y+height {
				switch content := cell.Content.(type) {
				case MouseableDrawableInteractive:
					content.MouseEvent(localX-x, localY-y, event)
				case Mouseable:
					content.MouseEvent(localX-x, localY-y, event)
				case MouseHandler:
					content.MouseEvent(localX-x, localY-y, event)
				}
			}
		}
	}
}

func (grid *Grid) reflow(ctx *Context) {
	grid.rowLayout = nil
	grid.columnLayout = nil
	flow := func(specs *[]GridSpec, layouts *[]gridLayout, extent int) {
		exact := 0
		weight := 0
		nweights := 0
		for _, spec := range *specs {
			if spec.Strategy == SIZE_EXACT {
				exact += spec.Size()
			} else if spec.Strategy == SIZE_WEIGHT {
				nweights += 1
				weight += spec.Size()
			}
		}
		offset := 0
		remainingExact := 0
		if weight > 0 {
			remainingExact = (extent - exact) % weight
		}
		for _, spec := range *specs {
			layout := gridLayout{Offset: offset}
			if spec.Strategy == SIZE_EXACT {
				layout.Size = spec.Size()
			} else if spec.Strategy == SIZE_WEIGHT {
				proportion := float64(spec.Size()) / float64(weight)
				size := proportion * float64(extent-exact)
				if remainingExact > 0 {
					extraExact := int(math.Ceil(proportion * float64(remainingExact)))
					layout.Size = int(math.Floor(size)) + extraExact
					remainingExact -= extraExact

				} else {
					layout.Size = int(math.Floor(size))
				}
			}
			offset += layout.Size
			*layouts = append(*layouts, layout)
		}
	}
	flow(&grid.rows, &grid.rowLayout, ctx.Height())
	flow(&grid.columns, &grid.columnLayout, ctx.Width())
	grid.invalid = false
}

func (grid *Grid) invalidateLayout() {
	grid.invalid = true
	grid.DoInvalidate(grid)
}

func (grid *Grid) Invalidate() {
	grid.invalidateLayout()
	grid.mutex.RLock()
	for _, cell := range grid.cells {
		cell.Content.Invalidate()
	}
	grid.mutex.RUnlock()
}

func (grid *Grid) AddChild(content Drawable) *GridCell {
	cell := &GridCell{
		RowSpan: 1,
		ColSpan: 1,
		Content: content,
	}
	grid.mutex.Lock()
	grid.cells = append(grid.cells, cell)
	grid.mutex.Unlock()
	cell.Content.OnInvalidate(grid.cellInvalidated)
	cell.invalid.Store(true)
	grid.invalidateLayout()
	return cell
}

func (grid *Grid) RemoveChild(content Drawable) {
	grid.mutex.Lock()
	for i, cell := range grid.cells {
		if cell.Content == content {
			grid.cells = append(grid.cells[:i], grid.cells[i+1:]...)
			break
		}
	}
	grid.mutex.Unlock()
	grid.invalidateLayout()
}

func (grid *Grid) cellInvalidated(drawable Drawable) {
	var cell *GridCell
	grid.mutex.RLock()
	for _, cell = range grid.cells {
		if cell.Content == drawable {
			break
		}
		cell = nil
	}
	grid.mutex.RUnlock()
	if cell == nil {
		panic(fmt.Errorf("Attempted to invalidate unknown cell"))
	}
	cell.invalid.Store(true)
	grid.DoInvalidate(grid)
}

func Const(i int) func() int {
	return func() int { return i }
}
