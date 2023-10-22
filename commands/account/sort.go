package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/sort"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Sort struct {
	Unused struct{} `opt:"-"`
	// these fields are only used for completion
	Reverse  bool     `opt:"-r"`
	Criteria []string `opt:"criteria" complete:"CompleteCriteria"`
}

func init() {
	register(Sort{})
}

func (Sort) Aliases() []string {
	return []string{"sort"}
}

var supportedCriteria = []string{
	"arrival",
	"cc",
	"date",
	"from",
	"read",
	"size",
	"subject",
	"to",
	"flagged",
}

func (*Sort) CompleteCriteria(arg string) []string {
	return commands.CompletionFromList(supportedCriteria, arg)
}

func (Sort) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected.")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Messages still loading.")
	}

	if c := store.Capabilities(); c != nil {
		if !c.Sort {
			return errors.New("Sorting is not available for this backend.")
		}
	}

	var err error
	var sortCriteria []*types.SortCriterion
	if len(args[1:]) == 0 {
		sortCriteria = acct.GetSortCriteria()
	} else {
		sortCriteria, err = sort.GetSortCriteria(args[1:])
		if err != nil {
			return err
		}
	}

	acct.SetStatus(state.Sorting(true))
	store.Sort(sortCriteria, func(msg types.WorkerMessage) {
		if _, ok := msg.(*types.Done); ok {
			acct.SetStatus(state.Sorting(false))
		}
	})
	return nil
}
