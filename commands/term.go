package commands

import (
	"os/exec"

	"github.com/riywo/loginshell"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Term struct {
	Cmd []string `opt:"..." required:"false"`
}

func init() {
	register(Term{})
}

func (Term) Aliases() []string {
	return []string{"terminal", "term"}
}

func (Term) Complete(args []string) []string {
	return nil
}

func (t Term) Execute(args []string) error {
	if len(t.Cmd) == 0 {
		shell, err := loginshell.Shell()
		if err != nil {
			return err
		}
		t.Cmd = []string{shell}
	}
	term, err := app.NewTerminal(exec.Command(t.Cmd[0], t.Cmd[1:]...))
	if err != nil {
		return err
	}
	tab := app.NewTab(term, t.Cmd[0])
	term.OnTitle = func(title string) {
		if title == "" {
			title = t.Cmd[0]
		}
		if tab.Name != title {
			tab.Name = title
			ui.Invalidate()
		}
	}
	term.OnClose = func(err error) {
		app.RemoveTab(term, false)
		if err != nil {
			app.PushError(err.Error())
		}
	}
	return nil
}
