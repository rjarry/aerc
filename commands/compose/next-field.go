package compose

import (
	"errors"
	"fmt"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type NextPrevField struct{}

func init() {
	register(NextPrevField{})
}

func (_ NextPrevField) Aliases() []string {
	return []string{"next-field", "prev-field"}
}

func (_ NextPrevField) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ NextPrevField) Execute(aerc *widgets.Aerc, args []string) error {
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

func nextPrevFieldUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s", cmd))
}
