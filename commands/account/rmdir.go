package account

import (
	"errors"
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt"
)

type RemoveDir struct {
	Force  bool   `opt:"-f"`
	Folder string `opt:"folder" complete:"CompleteFolder" required:"false"`
}

func init() {
	commands.Register(RemoveDir{})
}

func (RemoveDir) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (RemoveDir) Aliases() []string {
	return []string{"rmdir"}
}

func (RemoveDir) CompleteFolder(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return nil
	}
	return commands.FilterList(acct.Directories().List(), arg, opt.QuoteArg)
}

func (r RemoveDir) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	current := acct.Directories().SelectedDirectory()
	toRemove := current
	if r.Folder != "" {
		toRemove = acct.Directories().Directory(r.Folder)
		if toRemove == nil {
			return fmt.Errorf("No such directory: %s", r.Folder)
		}
	}

	role := toRemove.Role

	// Check for any messages in the directory.
	if role != models.QueryRole && toRemove.Exists > 0 && !r.Force {
		return errors.New("Refusing to remove non-empty directory; use -f")
	}

	if role == models.VirtualRole {
		return errors.New("Cannot remove a virtual node")
	}

	if toRemove != current {
		r.remove(acct, toRemove, func() {})
		return nil
	}

	curDir := current.Name
	var newDir string
	dirFound := false

	oldDir := acct.Directories().Previous()
	if oldDir != "" {
		present := false
		for _, dir := range acct.Directories().List() {
			if dir == oldDir {
				present = true
				break
			}
		}
		if oldDir != curDir && present {
			newDir = oldDir
			dirFound = true
		}
	}

	defaultDir := acct.AccountConfig().Default
	if !dirFound && defaultDir != curDir {
		for _, dir := range acct.Directories().List() {
			if defaultDir == dir {
				newDir = dir
				dirFound = true
				break
			}
		}
	}

	if !dirFound {
		for _, dir := range acct.Directories().List() {
			if dir != curDir {
				newDir = dir
				dirFound = true
				break
			}
		}
	}

	if !dirFound {
		return errors.New("No directory to move to afterwards!")
	}

	reopenCurrentDir := func() { acct.Directories().Open(curDir, "", 0, nil, false) }

	acct.Directories().Open(newDir, "", 0, func(msg types.WorkerMessage) {
		switch msg.(type) {
		case *types.Done:
			break
		case *types.Error:
			app.PushError("Could not change directory")
			reopenCurrentDir()
			return
		default:
			return
		}
		r.remove(acct, toRemove, reopenCurrentDir)
	}, false)

	return nil
}

func (r RemoveDir) remove(acct *app.AccountView, dir *models.Directory, onErr func()) {
	acct.Worker().PostAction(&types.RemoveDirectory{
		Directory: dir.Name,
		Quiet:     r.Force,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			app.PushStatus("Directory removed.", 10*time.Second)
		case *types.Error:
			app.PushError(msg.Error.Error())
			onErr()
		case *types.Unsupported:
			app.PushError(":rmdir is not supported by the backend.")
			onErr()
		}
	})
}
