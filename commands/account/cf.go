package account

import (
	"errors"
	"reflect"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt"
)

var history map[string]string

type ChangeFolder struct {
	Folder []string `opt:"..." complete:"CompleteFolder"`
}

func init() {
	history = make(map[string]string)
	register(ChangeFolder{})
}

func (ChangeFolder) Aliases() []string {
	return []string{"cf"}
}

func (*ChangeFolder) CompleteFolder(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return nil
	}
	return commands.FilterList(
		acct.Directories().List(), arg,
		func(s string) string {
			dir := acct.Directories().Directory(s)
			if dir != nil && dir.Role != models.QueryRole {
				s = opt.QuoteArg(s)
			}
			return s
		},
	)
}

func (c ChangeFolder) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	var target string

	notmuch, _ := handlers.GetHandlerForScheme("notmuch", new(types.Worker))
	switch {
	case reflect.TypeOf(notmuch) == reflect.TypeOf(acct.Worker().Backend):
		// notmuch query may have arguments that require quoting
		target = opt.QuoteArgs(c.Folder...).String()
	case len(c.Folder) == 1:
		target = c.Folder[0]
	default:
		return errors.New("Unexpected argument(s). Usage: cf <folder>")
	}

	previous := acct.Directories().Selected()

	if target == "-" {
		if dir, ok := history[acct.Name()]; ok {
			acct.Directories().Select(dir)
		} else {
			return errors.New("No previous folder to return to")
		}
	} else {
		acct.Directories().Select(target)
	}
	history[acct.Name()] = previous

	// reset store filtering if we switched folders
	store := acct.Store()
	if store != nil {
		store.ApplyClear()
		acct.SetStatus(state.SearchFilterClear())
	}
	return nil
}
