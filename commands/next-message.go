package commands

import (
	"errors"
	"fmt"
	"strconv"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	Register("next-message", NextPrevMessage)
	Register("prev-message", NextPrevMessage)
}

func nextPrevMessageUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [n]", cmd))
}

func NextPrevMessage(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return nextPrevMessageUsage(args[0])
	}
	var (
		n   int = 1
		err error
	)
	if len(args) > 1 {
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return nextPrevMessageUsage(args[0])
		}
	}
	acct := aerc.SelectedAccount()
	for ; n > 0; n-- {
		if args[0] == "prev-message" {
			acct.Messages().Prev()
		} else {
			acct.Messages().Next()
		}
	}
	return nil
}
