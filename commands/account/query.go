package account

import (
	"errors"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Query struct {
	Account string `opt:"-a" complete:"CompleteAccount" desc:"Account name."`
	Name    string `opt:"-n" desc:"Force name of virtual folder."`
	Force   bool   `opt:"-f" desc:"Replace existing query if any."`
	Query   string `opt:"..." complete:"CompleteNotmuch" desc:"Notmuch query."`
}

func init() {
	commands.Register(Query{})
}

func (Query) Description() string {
	return "Create a virtual folder using the specified notmuch query."
}

func (Query) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (Query) Aliases() []string {
	return []string{"query"}
}

func (Query) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, nil)
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

	if acct.AccountConfig().Backend != "notmuch" {
		return errors.New(":query is only available for notmuch accounts")
	}

	finalize := func(msg types.WorkerMessage) {
		handleDirOpenResponse(acct, msg)
	}

	name := q.Name
	if name == "" {
		name = q.Query
	}
	acct.Directories().Open(name, q.Query, 0*time.Second, finalize, q.Force)
	return nil
}

func (*Query) CompleteNotmuch(arg string) []string {
	return handleNotmuchComplete(arg)
}

var notmuch_search_terms = []string{
	"from:",
	"to:",
	"tag:",
	"date:",
	"attachment:",
	"mimetype:",
	"subject:",
	"body:",
	"id:",
	"thread:",
	"folder:",
	"path:",
}

func handleNotmuchComplete(arg string) []string {
	var found bool

	prefixes := []string{"from:", "to:"}
	for _, prefix := range prefixes {
		if arg, found = strings.CutPrefix(arg, prefix); found {
			return commands.FilterList(
				commands.GetAddress(arg), arg,
				func(v string) string { return prefix + v },
			)
		}
	}

	prefixes = []string{"tag:"}
	for _, prefix := range prefixes {
		if arg, found = strings.CutPrefix(arg, prefix); found {
			return commands.FilterList(
				commands.GetLabels(arg), arg,
				func(v string) string { return prefix + v },
			)
		}
	}

	prefixes = []string{"path:", "folder:"}
	dbPath := strings.TrimPrefix(app.SelectedAccount().AccountConfig().Source, "notmuch://")
	for _, prefix := range prefixes {
		if arg, found = strings.CutPrefix(arg, prefix); found {
			return commands.FilterList(
				commands.CompletePath(dbPath+arg, true), arg,
				func(v string) string { return prefix + strings.TrimPrefix(v, dbPath) },
			)
		}
	}

	return commands.FilterList(notmuch_search_terms, arg, nil)
}
