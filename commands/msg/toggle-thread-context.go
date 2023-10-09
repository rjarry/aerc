package msg

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type ToggleThreadContext struct{}

func init() {
	register(ToggleThreadContext{})
}

func (ToggleThreadContext) Aliases() []string {
	return []string{"toggle-thread-context"}
}

func (ToggleThreadContext) Complete(args []string) []string {
	return nil
}

func (ToggleThreadContext) Execute(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: toggle-entire-thread")
	}
	h := newHelper()
	store, err := h.store()
	if err != nil {
		return err
	}
	store.ToggleThreadContext()
	ui.Invalidate()
	return nil
}
