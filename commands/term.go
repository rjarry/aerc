package commands

import (
	"os/exec"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/widgets"

	"github.com/riywo/loginshell"
)

func init() {
	Register("term", Term)
}

func Term(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		shell, err := loginshell.Shell()
		if err != nil {
			return err
		}
		args = append(args, shell)
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
	tab := aerc.NewTab(grid, "Terminal")
	term.OnTitle = func(title string) {
		if title == "" {
			title = "Terminal"
		}
		tab.Name = title
		tab.Content.Invalidate()
	}
	return nil
}
