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

func (Term) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

// The help command is an alias for `term man` thus Term requires a simple func
func TermCore(aerc *app.Aerc, args []string) error {
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
	tab := aerc.NewTab(term, args[1])
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
		aerc.RemoveTab(term, false)
		if err != nil {
			aerc.PushError(err.Error())
		}
	}
	return nil
}

func (Term) Execute(aerc *app.Aerc, args []string) error {
	return TermCore(aerc, args)
}
