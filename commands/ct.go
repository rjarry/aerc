package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
)

type ChangeTab struct{}

func init() {
	register(ChangeTab{})
}

func (ChangeTab) Aliases() []string {
	return []string{"ct", "change-tab"}
}

func (ChangeTab) Complete(args []string) []string {
	if len(args) == 0 {
		return app.TabNames()
	}
	joinedArgs := strings.Join(args, " ")
	return FilterList(app.TabNames(), joinedArgs, "", app.SelectedAccountUiConfig().FuzzyComplete)
}

func (ChangeTab) Execute(args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: %s <tab>", args[0])
	}
	joinedArgs := strings.Join(args[1:], " ")
	if joinedArgs == "-" {
		ok := app.SelectPreviousTab()
		if !ok {
			return errors.New("No previous tab to return to")
		}
	} else {
		n, err := strconv.Atoi(joinedArgs)
		if err == nil {
			switch {
			case strings.HasPrefix(joinedArgs, "+"):
				for ; n > 0; n-- {
					app.NextTab()
				}
			case strings.HasPrefix(joinedArgs, "-"):
				for ; n < 0; n++ {
					app.PrevTab()
				}
			default:
				ok := app.SelectTabIndex(n)
				if !ok {
					return errors.New(
						"No tab with that index")
				}
			}
		} else {
			ok := app.SelectTab(joinedArgs)
			if !ok {
				return errors.New("No tab with that name")
			}
		}
	}
	return nil
}
