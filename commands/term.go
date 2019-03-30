package commands

import (
	"os/exec"
	"time"

	"git.sr.ht/~sircmpwn/aerc2/widgets"

	"github.com/gdamore/tcell"
	"github.com/riywo/loginshell"
)

func init() {
	register("term", Term)
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
	host := widgets.NewTermHost(term, aerc.Config())
	tab := aerc.NewTab(host, args[1])
	term.OnTitle = func(title string) {
		if title == "" {
			title = args[1]
		}
		tab.Name = title
		tab.Content.Invalidate()
	}
	term.OnClose = func(err error) {
		aerc.RemoveTab(host)
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
	}
	return nil
}
