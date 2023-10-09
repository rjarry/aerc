package commands

import (
	"fmt"
	"strconv"

	"git.sr.ht/~rjarry/aerc/app"
)

type NextPrevTab struct{}

func init() {
	register(NextPrevTab{})
}

func (NextPrevTab) Aliases() []string {
	return []string{"next-tab", "prev-tab"}
}

func (NextPrevTab) Complete(args []string) []string {
	return nil
}

func (NextPrevTab) Execute(args []string) error {
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
			app.PrevTab()
		} else {
			app.NextTab()
		}
	}
	app.UpdateStatus()
	return nil
}

func nextPrevTabUsage(cmd string) error {
	return fmt.Errorf("Usage: %s [n]", cmd)
}
