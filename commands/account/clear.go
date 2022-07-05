package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/lib/statusline"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Clear struct{}

func init() {
	register(Clear{})
}

func (Clear) Aliases() []string {
	return []string{"clear"}
}

func (Clear) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Clear) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	clearSelected := false
	opts, optind, err := getopt.Getopts(args, "s")
	if err != nil {
		return err
	}

	for _, opt := range opts {
		switch opt.Option {
		case 's':
			clearSelected = true
		}
	}

	if len(args) != optind {
		return errors.New("Usage: clear [-s]")
	}

	if clearSelected {
		defer store.Select(0)
	} else {
		store.SetReselect(store.Selected())
	}
	store.ApplyClear()
	acct.SetStatus(statusline.SearchFilterClear())

	return nil
}
