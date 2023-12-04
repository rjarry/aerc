package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type SelectMessage struct {
	Index int `opt:"n"`
}

func init() {
	commands.Register(SelectMessage{})
}

func (SelectMessage) Context() commands.CommandContext {
	return commands.ACCOUNT
}

func (SelectMessage) Aliases() []string {
	return []string{"select", "select-message"}
}

func (s SelectMessage) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if acct.Messages().Empty() {
		return nil
	}
	acct.Messages().Select(s.Index)
	return nil
}
