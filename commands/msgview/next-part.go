package msgview

import (
	"errors"
	"fmt"
	"strconv"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type NextPrevPart struct{}

func init() {
	register(NextPrevPart{})
}

func (_ NextPrevPart) Aliases() []string {
	return []string{"next-part", "prev-part"}
}

func (_ NextPrevPart) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ NextPrevPart) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return nextPrevPartUsage(args[0])
	}
	var (
		n   int = 1
		err error
	)
	if len(args) > 1 {
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return nextPrevPartUsage(args[0])
		}
	}
	mv, _ := aerc.SelectedTab().(*widgets.MessageViewer)
	for ; n > 0; n-- {
		if args[0] == "prev-part" {
			mv.PreviousPart()
		} else {
			mv.NextPart()
		}
	}
	return nil
}

func nextPrevPartUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [n]", cmd))
}
