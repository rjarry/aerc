package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type CheckMail struct{}

func init() {
	commands.Register(CheckMail{})
}

func (CheckMail) Description() string {
	return "Check for new mail on the selected account."
}

func (CheckMail) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
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
