package account

import (
	"errors"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type SearchFilter struct{}

func init() {
	register(SearchFilter{})
}

func (SearchFilter) Options() string {
	return "rubax:X:t:H:f:c:d:"
}

func (SearchFilter) Aliases() []string {
	return []string{"search", "filter"}
}

func (s SearchFilter) CompleteOption(
	r rune,
	search string,
) []string {
	var valid []string
	switch r {
	case 'x', 'X':
		valid = commands.GetFlagList()
	case 't', 'f', 'c':
		valid = commands.GetAddress(search)
	case 'd':
		valid = commands.GetDateList()
	default:
	}
	return commands.CompletionFromList(valid, []string{search})
}

func (SearchFilter) Complete(args []string) []string {
	return nil
}

func (SearchFilter) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	if args[0] == "filter" {
		if len(args[1:]) == 0 {
			return Clear{}.Execute([]string{"clear"})
		}
		acct.SetStatus(state.FilterActivity("Filtering..."), state.Search(""))
		store.SetFilter(args[1:])
		cb := func(msg types.WorkerMessage) {
			if _, ok := msg.(*types.Done); ok {
				acct.SetStatus(state.FilterResult(strings.Join(args, " ")))
				log.Tracef("Filter results: %v", store.Uids())
			}
		}
		store.Sort(store.GetCurrentSortCriteria(), cb)
	} else {
		acct.SetStatus(state.Search("Searching..."))
		cb := func(uids []uint32) {
			acct.SetStatus(state.Search(strings.Join(args, " ")))
			log.Tracef("Search results: %v", uids)
			store.ApplySearch(uids)
			// TODO: Remove when stores have multiple OnUpdate handlers
			ui.Invalidate()
		}
		store.Search(args, cb)
	}
	return nil
}
