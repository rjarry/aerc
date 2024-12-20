package msg

import (
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type ToggleThreads struct{}

func init() {
	commands.Register(ToggleThreads{})
}

func (ToggleThreads) Description() string {
	return "Toggle between message threading and the normal message list."
}

func (ToggleThreads) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (ToggleThreads) Aliases() []string {
	return []string{"toggle-threads"}
}

func (ToggleThreads) Execute(args []string) error {
	h := newHelper()
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
