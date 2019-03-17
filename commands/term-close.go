package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	Register("term-close", TermClose)
}

func TermClose(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: term-close")
	}
	grid, ok := aerc.SelectedTab().(*ui.Grid)
	if !ok {
		return errors.New("Error: not a terminal")
	}
	for _, child := range grid.Children() {
		if term, ok := child.(*widgets.Terminal); ok {
			term.Close(nil)
			return nil
		}
	}
	return errors.New("Error: not a terminal")
}
