package commands

import "git.sr.ht/~rjarry/aerc/lib/ui"

type Suspend struct{}

func init() {
	register(Suspend{})
}

func (Suspend) Aliases() []string {
	return []string{"suspend"}
}

func (Suspend) Complete(args []string) []string {
	return nil
}

func (Suspend) Execute(args []string) error {
	ui.QueueSuspend()
	return nil
}
