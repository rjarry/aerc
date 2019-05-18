package msgview

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("close", CommandClose)
}

func CommandClose(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: close")
	}
	mv, _ := aerc.SelectedTab().(*widgets.MessageViewer)
	aerc.RemoveTab(mv)
	return nil
}
