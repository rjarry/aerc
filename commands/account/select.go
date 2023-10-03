package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
)

type SelectMessage struct {
	Index int `opt:"n"`
}

func init() {
	register(SelectMessage{})
}

func (SelectMessage) Aliases() []string {
	return []string{"select", "select-message"}
}

func (SelectMessage) Complete(args []string) []string {
	return nil
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
