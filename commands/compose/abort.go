package compose

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("abort", CommandAbort)
}

func CommandAbort(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: abort")
	}
	composer, _ := aerc.SelectedTab().(*widgets.Composer)

	aerc.RemoveTab(composer)
	composer.Close()

	return nil
}
