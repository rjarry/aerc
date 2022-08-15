package compose

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type AccountSwitcher interface {
	SwitchAccount(*widgets.AccountView) error
}

type SwitchAccount struct{}

func init() {
	register(SwitchAccount{})
}

func (SwitchAccount) Aliases() []string {
	return []string{"switch-account"}
}

func (SwitchAccount) Complete(aerc *widgets.Aerc, args []string) []string {
	return aerc.AccountNames()
}

func (SwitchAccount) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		name := ""
		if acct := aerc.SelectedAccount(); acct != nil {
			name = fmt.Sprintf("Current account: %s. ", acct.Name())
		}
		return errors.New(name + "Usage: switch-account <account-name>")
	}

	switcher, ok := aerc.SelectedTabContent().(AccountSwitcher)
	if !ok {
		return errors.New("this tab cannot switch accounts")
	}

	if acct, err := aerc.Account(args[1]); err != nil {
		return err
	} else {
		return switcher.SwitchAccount(acct)
	}
}
