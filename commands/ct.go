package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type ChangeTab struct{}

func init() {
	register(ChangeTab{})
}

func (ChangeTab) Aliases() []string {
	return []string{"ct", "change-tab"}
}

func (ChangeTab) Complete(aerc *widgets.Aerc, args []string) []string {
	if len(args) == 0 {
		return aerc.TabNames()
	}
	joinedArgs := strings.Join(args, " ")
	return FilterList(aerc.TabNames(), joinedArgs, "", aerc.SelectedAccountUiConfig().FuzzyComplete)
}

func (ChangeTab) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: %s <tab>", args[0])
	}
	joinedArgs := strings.Join(args[1:], " ")
	if joinedArgs == "-" {
		ok := aerc.SelectPreviousTab()
		if !ok {
			return errors.New("No previous tab to return to")
		}
	} else {
		n, err := strconv.Atoi(joinedArgs)
		if err == nil {
			if strings.HasPrefix(joinedArgs, "+") {
				for ; n > 0; n-- {
					aerc.NextTab()
				}
			} else if strings.HasPrefix(joinedArgs, "-") {
				for ; n < 0; n++ {
					aerc.PrevTab()
				}
			} else {
				ok := aerc.SelectTabIndex(n)
				if !ok {
					return errors.New(
						"No tab with that index")
				}
			}
		} else {
			ok := aerc.SelectTab(joinedArgs)
			if !ok {
				return errors.New("No tab with that name")
			}
		}
	}
	return nil
}
