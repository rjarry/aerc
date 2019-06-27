package account

import (
	"errors"
	"fmt"
	"strconv"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type NextPrevFolder struct{}

func init() {
	register(NextPrevFolder{})
}

func (_ NextPrevFolder) Aliases() []string {
	return []string{"next-folder", "prev-folder"}
}

func (_ NextPrevFolder) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ NextPrevFolder) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return nextPrevFolderUsage(args[0])
	}
	var (
		n   int = 1
		err error
	)
	if len(args) > 1 {
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return nextPrevFolderUsage(args[0])
		}
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	for ; n > 0; n-- {
		if args[0] == "prev-folder" {
			acct.Directories().Prev()
		} else {
			acct.Directories().Next()
		}
	}
	return nil
}

func nextPrevFolderUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [n]", cmd))
}
