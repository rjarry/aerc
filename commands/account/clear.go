package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/state"
)

type Clear struct {
	Selected bool `opt:"-s"`
}

func init() {
	commands.Register(Clear{})
}

func (Clear) Context() commands.CommandContext {
	return commands.ACCOUNT
}

func (Clear) Aliases() []string {
	return []string{"clear"}
}

func (c Clear) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	if c.Selected {
		defer store.Select(0)
	}
	store.ApplyClear()
	acct.SetStatus(state.SearchFilterClear())

	return nil
}
