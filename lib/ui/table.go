package ui

import (
	"math"
	"regexp"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type Table struct {
	Columns []Column
	Rows    []Row
	Height  int
	// Optional callback that allows customizing the default drawing routine
	// of table rows. If true is returned, the default routine is skipped.
	CustomDraw func(t *Table, row int, c *Context) bool
	// Optional callback that allows returning a custom style for the row.
	GetRowStyle func(t *Table, row int) tcell.Style

	// true if at least one column has WIDTH_FIT
	autoFitWidths bool
	// if false, widths need to be computed before drawing
	widthsComputed bool
}

type Column struct {
	Offset    int
	Width     int
	Def       *config.ColumnDef
	Separator string
}

type Row struct {
	Cells []string
	Priv  interface{}
}

func NewTable(
	height int,
	columnDefs []*config.ColumnDef, separator string,
	customDraw func(*Table, int, *Context) bool,
	getRowStyle func(*Table, int) tcell.Style,
) Table {
	if customDraw == nil {
		customDraw = func(*Table, int, *Context) bool { return false }
	}
	if getRowStyle == nil {
		getRowStyle = func(*Table, int) tcell.Style {
			return tcell.StyleDefault
		}
	}
	columns := make([]Column, len(columnDefs))
	autoFitWidths := false
	for c, col := range columnDefs {
		if col.Flags.Has(config.WIDTH_FIT) {
			autoFitWidths = true
		}
		columns[c] = Column{Def: col}
		if c != len(columns)-1 {
			// set separator for all columns except the last one
			columns[c].Separator = separator
		}
	}
	return Table{
		Columns:       columns,
		Height:        height,
		CustomDraw:    customDraw,
		GetRowStyle:   getRowStyle,
		autoFitWidths: autoFitWidths,
	}
}

// add a row to the table, returns true when the table is full
func (t *Table) AddRow(cells []string, priv interface{}) bool {
	if len(cells) != len(t.Columns) {
		panic("invalid number of cells")
	}
	if len(t.Rows) >= t.Height {
		return true
	}
	t.Rows = append(t.Rows, Row{Cells: cells, Priv: priv})
	if t.autoFitWidths {
		t.widthsComputed = false
	}
	return len(t.Rows) >= t.Height
}

func (t *Table) computeWidths(width int) {
	contentMaxWidths := make([]int, len(t.Columns))
	if t.autoFitWidths {
		for _, row := range t.Rows {
			for c := range t.Columns {
				buf := parse.ParseANSI(row.Cells[c])
				if buf.Len() > contentMaxWidths[c] {
					contentMaxWidths[c] = buf.Len()
				}
			}
		}
	}

	nonFixed := width
	autoWidthCount := 0
	for c := range t.Columns {
		col := &t.Columns[c]
		switch {
		case col.Def.Flags.Has(config.WIDTH_FIT):
			col.Width = contentMaxWidths[c]
			// compensate for exact width columns
			col.Width += runewidth.StringWidth(col.Separator)
		case col.Def.Flags.Has(config.WIDTH_EXACT):
			col.Width = int(math.Round(col.Def.Width))
			// compensate for exact width columns
			col.Width += runewidth.StringWidth(col.Separator)
		case col.Def.Flags.Has(config.WIDTH_AUTO):
			col.Width = 0
			autoWidthCount += 1
		case col.Def.Flags.Has(config.WIDTH_FRACTION):
			col.Width = int(math.Round(float64(width) * col.Def.Width))
		}
		nonFixed -= col.Width
	}

	autoWidth := 0
	if autoWidthCount > 0 && nonFixed > 0 {
		autoWidth = nonFixed / autoWidthCount
		if autoWidth == 0 {
			autoWidth = 1
		}
	}

	offset := 0
	remain := width
	for c := range t.Columns {
		col := &t.Columns[c]
		if col.Def.Flags.Has(config.WIDTH_AUTO) && autoWidth > 0 {
			col.Width = autoWidth
			if nonFixed >= 2*autoWidth {
				nonFixed -= autoWidth
			}
		}
		if remain == 0 {
			// column is outside of screen
			col.Width = -1
		} else if col.Width > remain {
			// limit width to avoid overflow
			col.Width = remain
		}
		remain -= col.Width
		col.Offset = offset
		offset += col.Width
		// reserve room for separator
		col.Width -= runewidth.StringWidth(col.Separator)
	}
}

var metaCharsRegexp = regexp.MustCompile(`[\t\r\f\n\v]`)

func (col *Column) alignCell(cell string) string {
	cell = metaCharsRegexp.ReplaceAllString(cell, " ")
	buf := parse.ParseANSI(cell)
	width := buf.Len()

	switch {
	case col.Def.Flags.Has(config.ALIGN_LEFT):
		if width < col.Width {
			buf.PadRight(col.Width, ' ', tcell.StyleDefault)
			cell = buf.String()
		} else if width > col.Width {
			cell = buf.Truncate(col.Width, '…')
		}
	case col.Def.Flags.Has(config.ALIGN_CENTER):
		if width < col.Width {
			pad := col.Width - width
			buf.PadLeft(col.Width-(pad/2), ' ', tcell.StyleDefault)
			buf.PadRight(col.Width, ' ', tcell.StyleDefault)
			cell = buf.String()
		} else if width > col.Width {
			cell = buf.Truncate(col.Width, '…')
		}
	case col.Def.Flags.Has(config.ALIGN_RIGHT):
		if width < col.Width {
			buf.PadLeft(col.Width, ' ', tcell.StyleDefault)
			cell = buf.String()
		} else if width > col.Width {
			cell = buf.TruncateHead(col.Width, '…')
		}
	}

	return cell
}

func (t *Table) Draw(ctx *Context) {
	if !t.widthsComputed {
		t.computeWidths(ctx.Width())
		t.widthsComputed = true
	}
	for r, row := range t.Rows {
		if t.CustomDraw(t, r, ctx) {
			continue
		}
		for c, col := range t.Columns {
			if col.Width == -1 {
				// column overflows screen width
				continue
			}
			cell := col.alignCell(row.Cells[c])
			style := t.GetRowStyle(t, r)

			buf := parse.ParseANSI(cell)
			buf.ApplyAttrs(style)
			cell = buf.String()
			ctx.Printf(col.Offset, r, style, "%s%s", cell, col.Separator)
		}
	}
}
