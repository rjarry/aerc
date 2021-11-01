package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Connection struct{}

func init() {
	register(Connection{})
}

func (Connection) Aliases() []string {
	return []string{"connect", "disconnect"}
}

func (Connection) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Connection) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if args[0] == "connect" {
		acct.Worker().PostAction(&types.Connect{}, nil)
		acct.SetStatus("Connecting...")
	} else {
		acct.Worker().PostAction(&types.Disconnect{}, nil)
		acct.SetStatus("Disconnecting...")
	}
	return nil
}
