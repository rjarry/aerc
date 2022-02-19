package account

import (
	"errors"
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type SearchFilter struct{}

func init() {
	register(SearchFilter{})
}

func (SearchFilter) Aliases() []string {
	return []string{"search", "filter"}
}

func (SearchFilter) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (SearchFilter) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	var cb func([]uint32)
	if args[0] == "filter" {
		aerc.PushStatus("Filtering...", 10*time.Second)
		cb = func(uids []uint32) {
			aerc.PushStatus(fmt.Sprintf("Filter complete %s", args), 20*time.Second)
			acct.Logger().Printf("Filter results: %v", uids)
			store.ApplyFilter(uids)
		}
	} else {
		aerc.PushStatus("Searching...", 10*time.Second)
		cb = func(uids []uint32) {
			aerc.PushStatus(fmt.Sprintf("Search complete %s", args), 20*time.Second)
			acct.Logger().Printf("Search results: %v", uids)
			store.ApplySearch(uids)
			// TODO: Remove when stores have multiple OnUpdate handlers
			acct.Messages().Invalidate()
		}
	}
	store.Search(args, cb)
	return nil
}
