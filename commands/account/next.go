package account

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type NextPrevMsg struct{}

func init() {
	register(NextPrevMsg{})
}

func (_ NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (_ NextPrevMsg) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ NextPrevMsg) Execute(aerc *widgets.Aerc, args []string) error {
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
		if args[0] == "prev-message" || args[0] == "prev" {
			store := acct.Store()
			if store != nil {
				store.Prev()
			}
			acct.Messages().Scroll()
		} else {
			store := acct.Store()
			if store != nil {
				store.Next()
			}
			acct.Messages().Scroll()
		}
	}
	return nil
}

func nextPrevMessageUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [<n>[%%]]", cmd))
}
