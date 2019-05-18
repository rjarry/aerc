package compose

import (
	"errors"
	"fmt"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("next-field", NextPrevField)
	register("prev-field", NextPrevField)
}

func nextPrevFieldUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s", cmd))
}

func NextPrevField(aerc *widgets.Aerc, args []string) error {
	if len(args) > 2 {
		return nextPrevFieldUsage(args[0])
	}
	composer, _ := aerc.SelectedTab().(*widgets.Composer)
	if args[0] == "prev-field" {
		composer.PrevField()
	} else {
		composer.NextField()
	}
	return nil
}
