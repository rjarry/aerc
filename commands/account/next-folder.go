package account

import (
	"errors"
	"fmt"
	"strconv"

	"git.sr.ht/~rjarry/aerc/app"
)

type NextPrevFolder struct{}

func init() {
	register(NextPrevFolder{})
}

func (NextPrevFolder) Aliases() []string {
	return []string{"next-folder", "prev-folder"}
}

func (NextPrevFolder) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (NextPrevFolder) Execute(aerc *app.Aerc, args []string) error {
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
	if args[0] == "prev-folder" {
		acct.Directories().NextPrev(-n)
	} else {
		acct.Directories().NextPrev(n)
	}
	return nil
}

func nextPrevFolderUsage(cmd string) error {
	return fmt.Errorf("Usage: %s [n]", cmd)
}
