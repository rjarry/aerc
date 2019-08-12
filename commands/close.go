package commands

import (
	"errors"
	"fmt"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Close struct{}

func init() {
	register(Close{})
}

func (_ Close) Aliases() []string {
	return []string{"close", "abort"}
}

func (_ Close) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Close) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return errors.New("Usage: close [tab name]")
	} else if len(args) == 1 {
		return CloseTab(aerc, aerc.SelectedTabName())
	} else {
		tabname := args[1]
		for _, tab := range aerc.TabNames() {
			if tab == tabname {
				return CloseTab(aerc, tabname)
			}
		}
		return errors.New(fmt.Sprintf("Tab %s not found", tabname))
	}
	return nil
}

func CloseTab(aerc *widgets.Aerc, tabname string) error {
	curTabIndex := aerc.SelectedTabIndex()
	aerc.SelectTab(tabname)
	switch tab := aerc.SelectedTab().(type) {
	default:
		aerc.RemoveTab(tab)
		return nil
	case *widgets.Terminal:
		tab.Close(nil)
		return nil
	case *widgets.Composer:
		aerc.RemoveTab(tab)
		tab.Close()
		return nil
	case *widgets.AccountView:
		aerc.SelectTabIndex(curTabIndex)
		return errors.New("Cannot close account tab")
	}
}
