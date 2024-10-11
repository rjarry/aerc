package commands

import "git.sr.ht/~rjarry/aerc/lib/ui"

type Redraw struct{}

func init() {
	Register(Redraw{})
}

func (Redraw) Description() string {
	return "Force a full redraw of the screen."
}

func (Redraw) Context() CommandContext {
	return GLOBAL
}

func (Redraw) Aliases() []string {
	return []string{"redraw"}
}

func (Redraw) Execute(args []string) error {
	ui.QueueRefresh()
	return nil
}
