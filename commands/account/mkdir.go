package account

import (
	"context"
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt/v2"
)

type MakeDir struct {
	Folder string `opt:"folder" complete:"CompleteFolder" desc:"Folder name."`
}

func init() {
	commands.Register(MakeDir{})
}

func (MakeDir) Description() string {
	return "Create and change to a new folder."
}

func (MakeDir) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
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
	acct.Worker().PostAction(context.TODO(), &types.CreateDirectory{
		Directory: m.Folder,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			app.PushStatus("Directory created.", 10*time.Second)
			acct.Directories().Open(m.Folder, "", 0, nil, false)
		case *types.Error:
			app.PushError(msg.Error.Error())
		}
	})
	return nil
}
