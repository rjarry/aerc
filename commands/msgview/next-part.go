package msgview

import (
	"fmt"
	"strconv"

	"git.sr.ht/~rjarry/aerc/app"
)

type NextPrevPart struct{}

func init() {
	register(NextPrevPart{})
}

func (NextPrevPart) Aliases() []string {
	return []string{"next-part", "prev-part"}
}

func (NextPrevPart) Complete(args []string) []string {
	return nil
}

func (NextPrevPart) Execute(args []string) error {
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
	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
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
	return fmt.Errorf("Usage: %s [n]", cmd)
}
