package commands

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	// TODO: Move this command into a terminal-specific command set
	register("close", TermClose)
}

func TermClose(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: close")
	}
	thost, ok := aerc.SelectedTab().(*widgets.TermHost)
	if !ok {
		return errors.New("Error: not a terminal")
	}
	thost.Terminal().Close(nil)
	return nil
}
