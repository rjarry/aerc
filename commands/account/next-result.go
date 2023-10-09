package account

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type NextPrevResult struct{}

func init() {
	register(NextPrevResult{})
}

func (NextPrevResult) Aliases() []string {
	return []string{"next-result", "prev-result"}
}

func (NextPrevResult) Complete(args []string) []string {
	return nil
}

func (NextPrevResult) Execute(args []string) error {
	if len(args) > 1 {
		return nextPrevResultUsage(args[0])
	}
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

func nextPrevResultUsage(cmd string) error {
	return fmt.Errorf("Usage: %s [<n>[%%]]", cmd)
}
