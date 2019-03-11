package commands

import (
	"errors"
	"fmt"
	"strconv"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	Register("next-folder", NextPrevFolder)
	Register("prev-folder", NextPrevFolder)
}

func usage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [n]", cmd))
}

func NextPrevFolder(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return usage(args[0])
	}
	var (
		n   int = 1
		err error
	)
	if len(args) > 1 {
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return usage(args[0])
		}
	}
	acct := aerc.SelectedAccount()
	for ; n > 0; n-- {
		if args[0] == "prev-folder" {
			acct.Directories().Prev()
		} else {
			acct.Directories().Next()
		}
	}
	return nil
}
