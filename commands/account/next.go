package account

import (
	"errors"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type NextPrevMsg struct {
	Amount  int `opt:"n" default:"1" metavar:"<n>[%]" action:"ParseAmount"`
	Percent bool
}

func init() {
	register(NextPrevMsg{})
}

func (np *NextPrevMsg) ParseAmount(arg string) error {
	if strings.HasSuffix(arg, "%") {
		np.Percent = true
		arg = strings.TrimSuffix(arg, "%")
	}
	i, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return err
	}
	np.Amount = int(i)
	return nil
}

func (NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (NextPrevMsg) Complete(args []string) []string {
	return nil
}

func (np NextPrevMsg) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	n := np.Amount
	if np.Percent {
		n = int(float64(acct.Messages().Height()) * (float64(n) / 100.0))
	}
	if args[0] == "prev-message" || args[0] == "prev" {
		store := acct.Store()
		if store != nil {
			store.NextPrev(-n)
			ui.Invalidate()
		}
	} else {
		store := acct.Store()
		if store != nil {
			store.NextPrev(n)
			ui.Invalidate()
		}
	}
	return nil
}
