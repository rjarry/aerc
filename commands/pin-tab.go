package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type PinTab struct{}

func init() {
	Register(PinTab{})
}

func (PinTab) Description() string {
	return "Move the current tab to the left and mark it as pinned."
}

func (PinTab) Context() CommandContext {
	return GLOBAL
}

func (PinTab) Aliases() []string {
	return []string{"pin-tab", "unpin-tab"}
}

func (PinTab) Execute(args []string) error {
	switch args[0] {
	case "pin-tab":
		app.PinTab()
	case "unpin-tab":
		app.UnpinTab()
	}

	return nil
}
