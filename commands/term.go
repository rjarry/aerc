package commands

import (
	"errors"
	"os/exec"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	Register("term", Term)
}

func Term(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return errors.New("Usage: term [<command>]")
	}
	term, err := widgets.NewTerminal(exec.Command(args[1], args[2:]...))
	if err != nil {
		return err
	}
	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_EXACT, aerc.Config().Ui.SidebarWidth},
		{ui.SIZE_WEIGHT, 1},
	})
	grid.AddChild(term).At(0, 1)
	aerc.NewTab(grid, "Terminal")
	// TODO: update tab name when child process changes it
	return nil
}
