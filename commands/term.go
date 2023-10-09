package commands

import (
	"os/exec"

	"github.com/riywo/loginshell"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Term struct{}

func init() {
	register(Term{})
}

func (Term) Aliases() []string {
	return []string{"terminal", "term"}
}

func (Term) Complete(args []string) []string {
	return nil
}

// The help command is an alias for `term man` thus Term requires a simple func
func TermCore(args []string) error {
	if len(args) == 1 {
		shell, err := loginshell.Shell()
		if err != nil {
			return err
		}
		args = append(args, shell)
	}
	term, err := app.NewTerminal(exec.Command(args[1], args[2:]...))
	if err != nil {
		return err
	}
	tab := app.NewTab(term, args[1])
	term.OnTitle = func(title string) {
		if title == "" {
			title = args[1]
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

func (Term) Execute(args []string) error {
	return TermCore(args)
}
