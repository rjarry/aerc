package account

import (
	"errors"
	"reflect"
	"strings"
	"time"

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
	Account bool   `opt:"-a"`
	Folder  string `opt:"..." complete:"CompleteFolderAndNotmuch"`
}

func init() {
	history = make(map[string]string)
	commands.Register(ChangeFolder{})
}

func (ChangeFolder) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (ChangeFolder) Aliases() []string {
	return []string{"cf"}
}

func (c *ChangeFolder) CompleteFolderAndNotmuch(arg string) []string {
	var acct *app.AccountView

	args := opt.LexArgs(c.Folder)
	if c.Account {
		accountName, _ := args.ArgSafe(0)
		if args.Count() <= 1 && arg == accountName {
			return commands.FilterList(
				app.AccountNames(), arg, commands.QuoteSpace)
		}
		acct, _ = app.Account(accountName)
	} else {
		acct = app.SelectedAccount()
	}
	if acct == nil {
		return nil
	}
	retval := commands.FilterList(
		acct.Directories().List(), arg,
		func(s string) string {
			dir := acct.Directories().Directory(s)
			if dir != nil && dir.Role != models.QueryRole {
				s = opt.QuoteArg(s)
			}
			return s
		},
	)
	notmuch, _ := handlers.GetHandlerForScheme("notmuch", new(types.Worker))
	if reflect.TypeOf(notmuch) == reflect.TypeOf(acct.Worker().Backend) {
		notmuchcomps := handleNotmuchComplete(arg)
		for _, prefix := range notmuch_search_terms {
			if strings.HasPrefix(arg, prefix) {
				return notmuchcomps
			}
		}
		retval = append(retval, notmuchcomps...)

	}
	return retval
}

func (c ChangeFolder) Execute([]string) error {
	var target string
	var acct *app.AccountView

	args := opt.LexArgs(c.Folder)

	if c.Account {
		names, err := args.ShiftSafe(1)
		if err != nil {
			return errors.New("<account> is required. Usage: cf -a <account> <folder>")
		}
		acct, err = app.Account(names[0])
		if err != nil {
			return err
		}
	} else {
		acct = app.SelectedAccount()
		if acct == nil {
			return errors.New("No account selected")
		}
	}

	if args.Count() == 0 {
		return errors.New("<folder> is required. Usage: cf [-a <account>] <folder>")
	}

	notmuch, _ := handlers.GetHandlerForScheme("notmuch", new(types.Worker))
	if reflect.TypeOf(notmuch) == reflect.TypeOf(acct.Worker().Backend) {
		// With notmuch, :cf can change to a "dynamic folder" that
		// contains the result of a query. Preserve the entered
		// arguments verbatim.
		target = args.String()
	} else {
		if args.Count() != 1 {
			return errors.New("Unexpected argument(s). Usage: cf [-a <account>] <folder>")
		}
		target = args.Arg(0)
	}

	finalize := func(msg types.WorkerMessage) {
		handleDirOpenResponse(acct, msg)
	}

	if target == "-" {
		if dir, ok := history[acct.Name()]; ok {
			acct.Directories().Open(dir, "", 0*time.Second, finalize)
		} else {
			return errors.New("No previous folder to return to")
		}
	} else {
		acct.Directories().Open(target, "", 0*time.Second, finalize)
	}

	return nil
}

func handleDirOpenResponse(acct *app.AccountView, msg types.WorkerMessage) {
	// As we're waiting for the worker to report status we must run
	// the rest of the actions in this callback.
	switch msg := msg.(type) {
	case *types.Error:
		app.PushError(msg.Error.Error())
	case *types.Done:
		curAccount := app.SelectedAccount()
		previous := curAccount.Directories().Selected()
		history[curAccount.Name()] = previous
		// reset store filtering if we switched folders
		store := acct.Store()
		if store != nil {
			store.ApplyClear()
			acct.SetStatus(state.SearchFilterClear())
		}
		// focus account tab
		acct.Select()
	}
}
