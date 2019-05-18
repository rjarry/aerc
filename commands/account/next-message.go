package account

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("next-message", NextPrevMessage)
	register("prev-message", NextPrevMessage)
}

func nextPrevMessageUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [<n>[%%]]", cmd))
}

func NextPrevMessage(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return nextPrevMessageUsage(args[0])
	}
	var (
		n   int = 1
		err error
		pct bool
	)
	if len(args) > 1 {
		if strings.HasSuffix(args[1], "%") {
			pct = true
			args[1] = args[1][:len(args[1])-1]
		}
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return nextPrevMessageUsage(args[0])
		}
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if pct {
		n = int(float64(acct.Messages().Height()) * (float64(n) / 100.0))
	}
	for ; n > 0; n-- {
		if args[0] == "prev-message" {
			acct.Messages().Prev()
		} else {
			acct.Messages().Next()
		}
	}
	return nil
}
