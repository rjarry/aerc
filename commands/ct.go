package commands

import (
	"errors"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
)

type ChangeTab struct {
	Tab string `opt:"tab" complete:"CompleteTab"`
}

func init() {
	Register(ChangeTab{})
}

func (ChangeTab) Context() CommandContext {
	return GLOBAL
}

func (ChangeTab) Aliases() []string {
	return []string{"ct", "change-tab"}
}

func (*ChangeTab) CompleteTab(arg string) []string {
	return FilterList(app.TabNames(), arg, nil)
}

func (c ChangeTab) Execute(args []string) error {
	if c.Tab == "-" {
		ok := app.SelectPreviousTab()
		if !ok {
			return errors.New("No previous tab to return to")
		}
	} else {
		n, err := strconv.Atoi(c.Tab)
		if err == nil {
			if strings.HasPrefix(c.Tab, "+") || strings.HasPrefix(c.Tab, "-") {
				app.SelectTabAtOffset(n)
			} else {
				ok := app.SelectTabIndex(n)
				if !ok {
					return errors.New("No tab with that index")
				}
			}
		} else {
			ok := app.SelectTab(c.Tab)
			if !ok {
				return errors.New("No tab with that name")
			}
		}
	}
	return nil
}
