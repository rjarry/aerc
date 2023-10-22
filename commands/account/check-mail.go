package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
)

type CheckMail struct{}

func init() {
	register(CheckMail{})
}

func (CheckMail) Aliases() []string {
	return []string{"check-mail"}
}

func (CheckMail) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	acct.CheckMailReset()
	acct.CheckMail()
	return nil
}
