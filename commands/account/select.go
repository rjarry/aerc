package account

import (
	"errors"
	"strconv"

	"git.sr.ht/~rjarry/aerc/app"
)

type SelectMessage struct{}

func init() {
	register(SelectMessage{})
}

func (SelectMessage) Aliases() []string {
	return []string{"select", "select-message"}
}

func (SelectMessage) Complete(args []string) []string {
	return nil
}

func (SelectMessage) Execute(args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: :select-message <n>")
	}
	var (
		n   int = 1
		err error
	)
	if len(args) > 1 {
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return errors.New("Usage: :select-message <n>")
		}
	}
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if acct.Messages().Empty() {
		return nil
	}
	acct.Messages().Select(n)
	return nil
}
