package commands

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~sircmpwn/getopt"
)

type NewAccount struct{}

func init() {
	register(NewAccount{})
}

func (NewAccount) Aliases() []string {
	return []string{"new-account"}
}

func (NewAccount) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (NewAccount) Execute(aerc *app.Aerc, args []string) error {
	opts, _, err := getopt.Getopts(args, "t")
	if err != nil {
		return errors.New("Usage: new-account [-t]")
	}
	wizard := app.NewAccountWizard(aerc)
	for _, opt := range opts {
		if opt.Option == 't' {
			wizard.ConfigureTemporaryAccount(true)
		}
	}
	wizard.Focus(true)
	aerc.NewTab(wizard, "New account")
	return nil
}
