package account

import (
	"errors"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type MakeDir struct{}

func init() {
	register(MakeDir{})
}

func (MakeDir) Aliases() []string {
	return []string{"mkdir"}
}

func (MakeDir) Complete(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	name := strings.Join(args, " ")

	list := app.SelectedAccount().Directories().List()
	inboxes := make([]string, len(list))
	copy(inboxes, list)

	// remove inboxes that don't match and append the path separator to all
	// others
	for i := len(inboxes) - 1; i >= 0; i-- {
		if !strings.HasPrefix(inboxes[i], name) && name != "" {
			inboxes = append(inboxes[:i], inboxes[i+1:]...)
			continue
		}
		inboxes[i] += app.SelectedAccount().Worker().PathSeparator()
	}
	return inboxes
}

func (MakeDir) Execute(args []string) error {
	if len(args) == 0 {
		return errors.New("Usage: :mkdir <name>")
	}
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	name := strings.Join(args[1:], " ")
	acct.Worker().PostAction(&types.CreateDirectory{
		Directory: name,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			app.PushStatus("Directory created.", 10*time.Second)
			acct.Directories().Select(name)
		case *types.Error:
			app.PushError(msg.Error.Error())
		}
	})
	return nil
}
