package msgview

import (
	"errors"
	"fmt"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("toggle-headers", ToggleHeaders)
}

func toggleHeadersUsage(cmd string) error {
	return errors.New(fmt.Sprintf("Usage: %s", cmd))
}

func ToggleHeaders(aerc *widgets.Aerc, args []string) error {
	if len(args) > 1 {
		return toggleHeadersUsage(args[0])
	}
	mv, _ := aerc.SelectedTab().(*widgets.MessageViewer)
	mv.ToggleHeaders()
	return nil
}
