package account

import (
	"errors"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
)

type Split struct{}

func init() {
	register(Split{})
}

func (Split) Aliases() []string {
	return []string{"split", "vsplit", "hsplit"}
}

func (Split) Complete(args []string) []string {
	return nil
}

func (Split) Execute(args []string) error {
	if len(args) > 2 {
		return errors.New("Usage: [v|h]split n")
	}
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := app.SelectedAccount().Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	n := 0
	if acct.SplitSize() == 0 {
		if args[0] == "split" {
			n = app.SelectedAccount().Messages().Height() / 4
		} else {
			n = app.SelectedAccount().Messages().Width() / 2
		}
	}

	var err error
	if len(args) > 1 {
		delta := false
		if strings.HasPrefix(args[1], "+") || strings.HasPrefix(args[1], "-") {
			delta = true
		}
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return errors.New("Usage: [v|h]split n")
		}
		if delta {
			n = acct.SplitSize() + n
			acct.SetSplitSize(n)
			return nil
		}
	}
	if n == acct.SplitSize() {
		// Repeated commands of the same size have the effect of
		// toggling the split
		n = 0
	}
	if n < 0 {
		// Don't allow split to go negative
		n = 1
	}
	switch args[0] {
	case "split", "hsplit":
		return acct.Split(n)
	case "vsplit":
		return acct.Vsplit(n)
	}
	return nil
}
