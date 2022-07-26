package account

import (
	"errors"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/statusline"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
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

	if args[0] == "filter" {
		if len(args[1:]) == 0 {
			return Clear{}.Execute(aerc, []string{"clear"})
		}
		acct.SetStatus(statusline.FilterActivity("Filtering..."), statusline.Search(""))
		store.SetFilter(args[1:])
		cb := func(msg types.WorkerMessage) {
			if _, ok := msg.(*types.Done); ok {
				acct.SetStatus(statusline.FilterResult(strings.Join(args, " ")))
				logging.Infof("Filter results: %v", store.Uids())
			}
		}
		store.Sort(store.GetCurrentSortCriteria(), cb)
	} else {
		acct.SetStatus(statusline.Search("Searching..."))
		cb := func(uids []uint32) {
			acct.SetStatus(statusline.Search(strings.Join(args, " ")))
			logging.Infof("Search results: %v", uids)
			store.ApplySearch(uids)
			// TODO: Remove when stores have multiple OnUpdate handlers
			acct.Messages().Invalidate()
		}
		store.Search(args, cb)
	}
	return nil
}
