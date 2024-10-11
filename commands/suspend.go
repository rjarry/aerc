package commands

import "git.sr.ht/~rjarry/aerc/lib/ui"

type Suspend struct{}

func init() {
	Register(Suspend{})
}

func (Suspend) Description() string {
	return "Suspend the aerc process."
}

func (Suspend) Context() CommandContext {
	return GLOBAL
}

func (Suspend) Aliases() []string {
	return []string{"suspend"}
}

func (Suspend) Execute(args []string) error {
	ui.QueueSuspend()
	return nil
}
