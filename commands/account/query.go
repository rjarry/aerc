package account

import (
	"errors"
	"reflect"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Query struct {
	Account string `opt:"-a" complete:"CompleteAccount"`
	Name    string `opt:"-n"`
	Query   string `opt:"..."`
}

func init() {
	commands.Register(Query{})
}

func (Query) Context() commands.CommandContext {
	return commands.ACCOUNT
}

func (Query) Aliases() []string {
	return []string{"query"}
}

func (Query) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, commands.QuoteSpace)
}

func (q Query) Execute([]string) error {
	var acct *app.AccountView

	if q.Account == "" {
		acct = app.SelectedAccount()
		if acct == nil {
			return errors.New("No account selected")
		}
	} else {
		var err error
		acct, err = app.Account(q.Account)
		if err != nil {
			return err
		}
	}

	notmuch, _ := handlers.GetHandlerForScheme("notmuch", new(types.Worker))
	if reflect.TypeOf(notmuch) != reflect.TypeOf(acct.Worker().Backend) {
		return errors.New(":query is only available for notmuch accounts")
	}

	finalize := func(msg types.WorkerMessage) {
		handleDirOpenResponse(acct, msg)
	}

	name := q.Name
	if name == "" {
		name = q.Query
	}
	acct.Directories().Open(name, q.Query, 0*time.Second, finalize)
	return nil
}
