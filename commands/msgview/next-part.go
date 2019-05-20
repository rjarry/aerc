package msgview

import (
	"errors"
	"fmt"
	"strconv"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("next-part", NextPrevPart)
	register("prev-part", NextPrevPart)
}

func nextPrevPartUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s [n]", cmd))
}

func NextPrevPart(aerc *widgets.Aerc, args []string) error {
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
