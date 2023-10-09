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

func (NewAccount) Complete(args []string) []string {
	return nil
}

func (NewAccount) Execute(args []string) error {
	opts, _, err := getopt.Getopts(args, "t")
	if err != nil {
		return errors.New("Usage: new-account [-t]")
	}
	wizard := app.NewAccountWizard()
	for _, opt := range opts {
		if opt.Option == 't' {
			wizard.ConfigureTemporaryAccount(true)
		}
	}
	wizard.Focus(true)
	app.NewTab(wizard, "New account")
	return nil
}
