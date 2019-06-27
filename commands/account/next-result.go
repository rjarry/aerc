package account

import (
	"errors"
	"fmt"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type NextPrevResult struct{}

func init() {
	register(NextPrevResult{})
}

func (_ NextPrevResult) Aliases() []string {
	return []string{"next-result", "prev-result"}
}

func (_ NextPrevResult) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ NextPrevResult) Execute(aerc *widgets.Aerc, args []string) error {
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
		acct.Messages().Scroll()
	} else {
		store := acct.Store()
		if store != nil {
			store.NextResult()
		}
		acct.Messages().Scroll()
	}
	return nil
}

func nextPrevResultUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [<n>[%%]]", cmd))
}
