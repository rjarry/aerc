package account

import (
	"errors"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Clear struct{}

func init() {
	register(Clear{})
}

func (_ Clear) Aliases() []string {
	return []string{"clear"}
}

func (_ Clear) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Clear) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	store.ApplyClear()
	aerc.SetStatus("Clear complete.")
	return nil
}
