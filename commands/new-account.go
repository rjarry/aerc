package commands

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type NewAccount struct{}

func init() {
	register(NewAccount{})
}

func (NewAccount) Aliases() []string {
	return []string{"new-account"}
}

func (NewAccount) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (NewAccount) Execute(aerc *widgets.Aerc, args []string) error {
	opts, _, err := getopt.Getopts(args, "t")
	if err != nil {
		return errors.New("Usage: new-account [-t]")
	}
	wizard := widgets.NewAccountWizard(aerc.Config(), aerc)
	for _, opt := range opts {
		if opt.Option == 't' {
			wizard.ConfigureTemporaryAccount(true)
		}
	}
	aerc.NewTab(wizard, "New account")
	return nil
}
