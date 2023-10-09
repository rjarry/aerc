package msg

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type ToggleThreads struct{}

func init() {
	register(ToggleThreads{})
}

func (ToggleThreads) Aliases() []string {
	return []string{"toggle-threads"}
}

func (ToggleThreads) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (ToggleThreads) Execute(aerc *app.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: toggle-threads")
	}
	h := newHelper(aerc)
	acct, err := h.account()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	store.SetThreadedView(!store.ThreadedView())
	acct.SetStatus(state.Threading(store.ThreadedView()))
	ui.Invalidate()
	return nil
}
