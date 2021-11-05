package account

import (
	"errors"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type MakeDir struct{}

func init() {
	register(MakeDir{})
}

func (MakeDir) Aliases() []string {
	return []string{"mkdir"}
}

func (MakeDir) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (MakeDir) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 0 {
		return errors.New("Usage: :mkdir <name>")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	name := strings.Join(args[1:], " ")
	acct.Worker().PostAction(&types.CreateDirectory{
		Directory: name,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Directory created.", 10*time.Second)
			acct.Directories().Select(name)
		case *types.Error:
			aerc.PushError(msg.Error.Error())
		}
	})
	return nil
}
