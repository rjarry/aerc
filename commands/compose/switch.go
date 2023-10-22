package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type AccountSwitcher interface {
	SwitchAccount(*app.AccountView) error
}

type SwitchAccount struct {
	Next    bool   `opt:"-n"`
	Prev    bool   `opt:"-p"`
	Account string `opt:"account" required:"false" complete:"CompleteAccount"`
}

func init() {
	register(SwitchAccount{})
}

func (SwitchAccount) Aliases() []string {
	return []string{"switch-account"}
}

func (*SwitchAccount) CompleteAccount(arg string) []string {
	return commands.CompletionFromList(app.AccountNames(), arg)
}

func (s SwitchAccount) Execute(args []string) error {
	if !s.Prev && !s.Next && s.Account == "" {
		return errors.New("Usage: switch-account -n | -p | <account-name>")
	}

	switcher, ok := app.SelectedTabContent().(AccountSwitcher)
	if !ok {
		return errors.New("this tab cannot switch accounts")
	}

	var acct *app.AccountView
	var err error

	switch {
	case s.Prev:
		acct, err = app.PrevAccount()
	case s.Next:
		acct, err = app.NextAccount()
	default:
		acct, err = app.Account(s.Account)
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
