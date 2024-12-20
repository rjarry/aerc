package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type NewAccount struct {
	Temp bool `opt:"-t" desc:"Create a temporary account."`
}

func init() {
	Register(NewAccount{})
}

func (NewAccount) Description() string {
	return "Start the new account wizard."
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
