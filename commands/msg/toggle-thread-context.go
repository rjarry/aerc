package msg

import (
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type ToggleThreadContext struct{}

func init() {
	register(ToggleThreadContext{})
}

func (ToggleThreadContext) Aliases() []string {
	return []string{"toggle-thread-context"}
}

func (ToggleThreadContext) Execute(args []string) error {
	h := newHelper()
	store, err := h.store()
	if err != nil {
		return err
	}
	store.ToggleThreadContext()
	ui.Invalidate()
	return nil
}
