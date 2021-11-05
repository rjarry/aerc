package account

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type NextPrevResult struct{}

func init() {
	register(NextPrevResult{})
}

func (NextPrevResult) Aliases() []string {
	return []string{"next-result", "prev-result"}
}

func (NextPrevResult) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (NextPrevResult) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) > 1 {
		return nextPrevResultUsage(args[0])
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if args[0] == "prev-result" {
		store := acct.Store()
		if store != nil {
			store.PrevResult()
		}
		acct.Messages().Invalidate()
	} else {
		store := acct.Store()
		if store != nil {
			store.NextResult()
		}
		acct.Messages().Invalidate()
	}
	return nil
}

func nextPrevResultUsage(cmd string) error {
	return fmt.Errorf("Usage: %s [<n>[%%]]", cmd)
}
