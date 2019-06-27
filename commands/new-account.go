package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type NewAccount struct{}

func init() {
	register(NewAccount{})
}

func (_ NewAccount) Aliases() []string {
	return []string{"new-account"}
}

func (_ NewAccount) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ NewAccount) Execute(aerc *widgets.Aerc, args []string) error {
	opts, _, err := getopt.Getopts(args, "t")
	if err != nil {
		return errors.New("Usage: new-account [-t]")
	}
	wizard := widgets.NewAccountWizard(aerc.Config(), aerc)
	for _, opt := range opts {
		switch opt.Option {
		case 't':
			wizard.ConfigureTemporaryAccount(true)
		}
	}
	aerc.NewTab(wizard, "New account")
	return nil
}
