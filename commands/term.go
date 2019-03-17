package commands

import (
	"os/exec"
	"time"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/widgets"

	"github.com/gdamore/tcell"
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
	tab := aerc.NewTab(grid, args[1])
	term.OnTitle = func(title string) {
		if title == "" {
			title = args[1]
		}
		tab.Name = title
		tab.Content.Invalidate()
	}
	term.OnClose = func(err error) {
		aerc.RemoveTab(grid)
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorRed, tcell.ColorWhite)
		}
	}
	return nil
}
