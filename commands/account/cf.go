package account

import (
	"errors"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt/v2"
)

type ChangeFolder struct {
	Account bool   `opt:"-a"`
	Folder  string `opt:"..." complete:"CompleteFolderAndNotmuch"`
}

func init() {
	commands.Register(ChangeFolder{})
}

func (ChangeFolder) Description() string {
	return "Change the folder shown in the message list."
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
	if acct.AccountConfig().Backend == "notmuch" {
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

	if acct.AccountConfig().Backend == "notmuch" {
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

	dirlist := acct.Directories()
	if dirlist == nil {
		return errors.New("No directory list found")
	}

	if target == "-" {
		dir := dirlist.Previous()
		if dir != "" {
			target = dir
		} else {
			return errors.New("No previous folder to return to")
		}
	}

	dirlist.Open(target, "", 0*time.Second, finalize, false)

	return nil
}

func handleDirOpenResponse(acct *app.AccountView, msg types.WorkerMessage) {
	// As we're waiting for the worker to report status we must run
	// the rest of the actions in this callback.
	switch msg := msg.(type) {
	case *types.Error:
		app.PushError(msg.Error.Error())
	case *types.Done:
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
