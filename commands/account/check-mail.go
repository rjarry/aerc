package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type CheckMail struct{}

func init() {
	register(CheckMail{})
}

func (CheckMail) Aliases() []string {
	return []string{"check-mail"}
}

func (CheckMail) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (CheckMail) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	acct.CheckMailReset()
	acct.CheckMail()
	return nil
}
