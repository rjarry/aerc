package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type NewAccount struct {
	Temp bool `opt:"-t"`
}

func init() {
	Register(NewAccount{})
}

func (NewAccount) Context() CommandContext {
	return GLOBAL
}

func (NewAccount) Aliases() []string {
	return []string{"new-account"}
}

func (n NewAccount) Execute(args []string) error {
	wizard := app.NewAccountWizard()
	wizard.ConfigureTemporaryAccount(n.Temp)
	wizard.Focus(true)
	app.NewTab(wizard, "New account")
	return nil
}
