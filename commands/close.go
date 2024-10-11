package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type Close struct{}

func init() {
	Register(Close{})
}

func (Close) Description() string {
	return "Close the focused tab."
}

func (Close) Context() CommandContext {
	return MESSAGE_VIEWER | TERMINAL
}

func (Close) Aliases() []string {
	return []string{"close"}
}

func (Close) Execute([]string) error {
	app.RemoveTab(app.SelectedTabContent(), true)
	return nil
}
