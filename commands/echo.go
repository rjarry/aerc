package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type Echo struct {
	Template string `opt:"..." required:"false"`
}

func init() {
	Register(Echo{})
}

func (Echo) Aliases() []string {
	return []string{"echo"}
}

func (Echo) Context() CommandContext {
	return GLOBAL
}

func (e Echo) Execute(args []string) error {
	app.PushSuccess(e.Template)
	return nil
}
