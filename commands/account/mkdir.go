package account

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt"
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
	sep := app.SelectedAccount().Worker().PathSeparator()
	return commands.FilterList(
		acct.Directories().List(), arg,
		func(s string) string {
			return opt.QuoteArg(s) + sep
		},
	)
}

func (m MakeDir) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	previous := acct.SelectedDirectory()
	acct.Worker().PostAction(&types.CreateDirectory{
		Directory: m.Folder,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			app.PushStatus("Directory created.", 10*time.Second)
			history[acct.Name()] = previous
			acct.Directories().Open(m.Folder, 0, nil)
		case *types.Error:
			app.PushError(msg.Error.Error())
		}
	})
	return nil
}
