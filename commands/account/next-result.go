package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type NextPrevResult struct{}

func init() {
	commands.Register(NextPrevResult{})
}

func (NextPrevResult) Context() commands.CommandContext {
	return commands.ACCOUNT
}

func (NextPrevResult) Aliases() []string {
	return []string{"next-result", "prev-result"}
}

func (NextPrevResult) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if args[0] == "prev-result" {
		store := acct.Store()
		if store != nil {
			store.PrevResult()
		}
		ui.Invalidate()
	} else {
		store := acct.Store()
		if store != nil {
			store.NextResult()
		}
		ui.Invalidate()
	}
	return nil
}
