package msg

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type ToggleThreadContext struct{}

func init() {
	register(ToggleThreadContext{})
}

func (ToggleThreadContext) Aliases() []string {
	return []string{"toggle-thread-context"}
}

func (ToggleThreadContext) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (ToggleThreadContext) Execute(aerc *app.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: toggle-entire-thread")
	}
	h := newHelper(aerc)
	store, err := h.store()
	if err != nil {
		return err
	}
	store.ToggleThreadContext()
	ui.Invalidate()
	return nil
}
