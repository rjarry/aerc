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
	register(ChangeTab{})
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
			switch {
			case strings.HasPrefix(c.Tab, "+"):
				for ; n > 0; n-- {
					app.NextTab()
				}
			case strings.HasPrefix(c.Tab, "-"):
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
			ok := app.SelectTab(c.Tab)
			if !ok {
				return errors.New("No tab with that name")
			}
		}
	}
	return nil
}
