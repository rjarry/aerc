package account

import (
	"errors"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("search", SearchFilter)
	//register("filter", SearchFilter) // TODO
}

func SearchFilter(aerc *widgets.Aerc, args []string) error {
	var (
		criteria *imap.SearchCriteria = imap.NewSearchCriteria()
	)

	opts, optind, err := getopt.Getopts(args, "ruH:")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'r':
			criteria.WithFlags = append(criteria.WithFlags, imap.SeenFlag)
		case 'u':
			criteria.WithoutFlags = append(criteria.WithoutFlags, imap.SeenFlag)
		case 'H':
			// TODO
		}
	}
	for _, arg := range args[optind:] {
		criteria.Header.Add("Subject", arg)
	}

	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	aerc.SetStatus("Searching...")
	store.Search(criteria, func(uids []uint32) {
		aerc.SetStatus("Search complete.")
		acct.Logger().Printf("Search results: %v", uids)
		store.ApplySearch(uids)
		// TODO: Remove when stores have multiple OnUpdate handlers
		acct.Messages().Scroll()
	})
	return nil
}
