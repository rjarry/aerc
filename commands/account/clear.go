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

func (Clear) Description() string {
	return "Clear the current search or filter criteria."
}

func (Clear) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
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
		defer store.Select("")
	}
	store.ApplyClear()
	acct.SetStatus(state.SearchFilterClear())

	return nil
}
