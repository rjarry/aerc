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

func (NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (NextPrevMsg) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

	var err, n, pct = ParseNextPrevMessage(args)
func (NextPrevMsg) Execute(aerc *widgets.Aerc, args []string) error {
	if err != nil {
		return err
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	return ExecuteNextPrevMessage(args, acct, pct, n)
}

func ParseNextPrevMessage(args []string) (error, int, bool) {
	if len(args) > 2 {
		return nextPrevMessageUsage(args[0]), 0, false
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
			return nextPrevMessageUsage(args[0]), 0, false
		}
	}
	return nil, n, pct
}

func ExecuteNextPrevMessage(args []string, acct *widgets.AccountView, pct bool, n int) error {
	if pct {
		n = int(float64(acct.Messages().Height()) * (float64(n) / 100.0))
	}
	if args[0] == "prev-message" || args[0] == "prev" {
		store := acct.Store()
		if store != nil {
			store.NextPrev(-n)
			acct.Messages().Scroll()
		}
	} else {
		store := acct.Store()
		if store != nil {
			store.NextPrev(n)
			acct.Messages().Scroll()
		}
	}
	return nil
}

func nextPrevMessageUsage(cmd string) error {
	return fmt.Errorf("Usage: %s [<n>[%%]]", cmd)
}
