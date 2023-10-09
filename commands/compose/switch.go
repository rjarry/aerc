package compose

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~sircmpwn/getopt"
)

type AccountSwitcher interface {
	SwitchAccount(*app.AccountView) error
}

type SwitchAccount struct{}

func init() {
	register(SwitchAccount{})
}

func (SwitchAccount) Aliases() []string {
	return []string{"switch-account"}
}

func (SwitchAccount) Complete(aerc *app.Aerc, args []string) []string {
	return aerc.AccountNames()
}

func (SwitchAccount) Execute(aerc *app.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, "np")
	if err != nil {
		return err
	}
	var next, prev bool
	for _, opt := range opts {
		switch opt.Option {
		case 'n':
			next = true
			prev = false
		case 'p':
			next = false
			prev = true
		}
	}
	posargs := args[optind:]
	// NOT ((prev || next) XOR (len(posargs) == 1))
	if (prev || next) == (len(posargs) == 1) {
		name := ""
		if acct := aerc.SelectedAccount(); acct != nil {
			name = fmt.Sprintf("Current account: %s. ", acct.Name())
		}
		return errors.New(name + "Usage: switch-account [-np] <account-name>")
	}

	switcher, ok := aerc.SelectedTabContent().(AccountSwitcher)
	if !ok {
		return errors.New("this tab cannot switch accounts")
	}

	var acct *app.AccountView

	switch {
	case prev:
		acct, err = aerc.PrevAccount()
	case next:
		acct, err = aerc.NextAccount()
	default:
		acct, err = aerc.Account(posargs[0])
	}
	if err != nil {
		return err
	}
	if err = switcher.SwitchAccount(acct); err != nil {
		return err
	}
	acct.UpdateStatus()

	return nil
}
