package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type NextPrevFolder struct {
	Offset int `opt:"n" default:"1"`
}

func init() {
	commands.Register(NextPrevFolder{})
}

func (NextPrevFolder) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (NextPrevFolder) Aliases() []string {
	return []string{"next-folder", "prev-folder"}
}

func (np NextPrevFolder) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if args[0] == "prev-folder" {
		acct.Directories().NextPrev(-np.Offset)
	} else {
		acct.Directories().NextPrev(np.Offset)
	}
	return nil
}
