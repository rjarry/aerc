package commands

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type PinTab struct{}

func init() {
	register(PinTab{})
}

func (PinTab) Aliases() []string {
	return []string{"pin-tab", "unpin-tab"}
}

func (PinTab) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (PinTab) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: %s", args[0])
	}

	switch args[0] {
	case "pin-tab":
		aerc.PinTab()
	case "unpin-tab":
		aerc.UnpinTab()
	}

	return nil
}
