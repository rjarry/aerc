package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Connection struct{}

func init() {
	register(Connection{})
}

func (Connection) Aliases() []string {
	return []string{"connect", "disconnect"}
}

func (Connection) Complete(args []string) []string {
	return nil
}

func (Connection) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	cb := func(msg types.WorkerMessage) {
		acct.SetStatus(state.ConnectionActivity(""))
	}
	if args[0] == "connect" {
		acct.Worker().PostAction(&types.Connect{}, cb)
		acct.SetStatus(state.ConnectionActivity("Connecting..."))
	} else {
		acct.Worker().PostAction(&types.Disconnect{}, cb)
		acct.SetStatus(state.ConnectionActivity("Disconnecting..."))
	}
	return nil
}
