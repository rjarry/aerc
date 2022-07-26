package account

import (
	"errors"
	"strings"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/sort"
	"git.sr.ht/~rjarry/aerc/lib/statusline"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Sort struct{}

func init() {
	register(Sort{})
}

func (Sort) Aliases() []string {
	return []string{"sort"}
}

func (Sort) Complete(aerc *widgets.Aerc, args []string) []string {
	supportedCriteria := []string{
		"arrival",
		"cc",
		"date",
		"from",
		"read",
		"size",
		"subject",
		"to",
	}
	if len(args) == 0 {
		return supportedCriteria
	}
	last := args[len(args)-1]
	var completions []string
	currentPrefix := strings.Join(args, " ") + " "
	// if there is a completed criteria or option then suggest all again
	for _, criteria := range append(supportedCriteria, "-r") {
		if criteria == last {
			for _, criteria := range supportedCriteria {
				completions = append(completions, currentPrefix+criteria)
			}
			return completions
		}
	}

	currentPrefix = strings.Join(args[:len(args)-1], " ")
	if len(args) > 1 {
		currentPrefix += " "
	}
	// last was beginning an option
	if last == "-" {
		return []string{currentPrefix + "-r"}
	}
	// the last item is not complete
	completions = commands.FilterList(supportedCriteria, last, currentPrefix,
		aerc.SelectedAccountUiConfig().FuzzyComplete)
	return completions
}

func (Sort) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
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

	acct.SetStatus(statusline.Sorting(true))
	store.Sort(sortCriteria, func(msg types.WorkerMessage) {
		if _, ok := msg.(*types.Done); ok {
			acct.SetStatus(statusline.Sorting(false))
		}
	})
	return nil
}
