package account

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type RemoveDir struct {
	Force bool `opt:"-f"`
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

func (r RemoveDir) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	var role models.Role
	if d := acct.Directories().SelectedDirectory(); d != nil {
		role = d.Role
	}

	// Check for any messages in the directory.
	if role != models.QueryRole && !acct.Messages().Empty() && !r.Force {
		return errors.New("Refusing to remove non-empty directory; use -f")
	}

	if role == models.VirtualRole {
		return errors.New("Cannot remove a virtual node")
	}

	curDir := acct.SelectedDirectory()
	var newDir string
	dirFound := false

	if oldDir, ok := history[acct.Name()]; ok {
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

	reopenCurrentDir := func() { acct.Directories().Open(curDir, "", 0, nil) }

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
		acct.Worker().PostAction(&types.RemoveDirectory{
			Directory: curDir,
			Quiet:     r.Force,
		}, func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				app.PushStatus("Directory removed.", 10*time.Second)
			case *types.Error:
				app.PushError(msg.Error.Error())
				reopenCurrentDir()
			case *types.Unsupported:
				app.PushError(":rmdir is not supported by the backend.")
				reopenCurrentDir()
			}
		})
	})

	return nil
}
