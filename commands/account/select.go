package account

import (
	"errors"
	"strconv"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type SelectMessage struct{}

func init() {
	register(SelectMessage{})
}

func (_ SelectMessage) Aliases() []string {
	return []string{"select", "select-message"}
}

func (_ SelectMessage) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ SelectMessage) Execute(aerc *widgets.Aerc, args []string) error {
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
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if acct.Messages().Empty() {
		return nil
	}
	acct.Messages().Select(n)
	return nil
}
