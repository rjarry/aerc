package commands

import (
	"errors"
	"fmt"
	"strconv"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	register("next-tab", NextPrevTab)
	register("prev-tab", NextPrevTab)
}

func nextPrevTabUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [n]", cmd))
}

func NextPrevTab(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return nextPrevTabUsage(args[0])
	}
	var (
		n   int = 1
		err error
	)
	if len(args) > 1 {
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return nextPrevTabUsage(args[0])
		}
	}
	for ; n > 0; n-- {
		if args[0] == "prev-tab" {
			aerc.PrevTab()
		} else {
			aerc.NextTab()
		}
	}
	return nil
}
