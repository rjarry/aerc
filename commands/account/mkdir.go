package account

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type MakeDir struct {
	Folder string `opt:"folder" complete:"CompleteFolder"`
}

func init() {
	register(MakeDir{})
}

func (MakeDir) Aliases() []string {
	return []string{"mkdir"}
}

func (*MakeDir) CompleteFolder(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return nil
	}
	return commands.FilterList(
		acct.Directories().List(), arg, "",
		app.SelectedAccount().Worker().PathSeparator(),
		app.SelectedAccountUiConfig().FuzzyComplete)
}

func (m MakeDir) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	acct.Worker().PostAction(&types.CreateDirectory{
		Directory: m.Folder,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			app.PushStatus("Directory created.", 10*time.Second)
			acct.Directories().Select(m.Folder)
		case *types.Error:
			app.PushError(msg.Error.Error())
		}
	})
	return nil
}
