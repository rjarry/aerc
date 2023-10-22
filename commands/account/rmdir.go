package account

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type RemoveDir struct {
	Force bool `opt:"-f"`
}

func init() {
	register(RemoveDir{})
}

func (RemoveDir) Aliases() []string {
	return []string{"rmdir"}
}

func (r RemoveDir) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	// Check for any messages in the directory.
	if !acct.Messages().Empty() && !r.Force {
		return errors.New("Refusing to remove non-empty directory; use -f")
	}

	curDir := acct.SelectedDirectory()
	var newDir string
	dirFound := false

	if oldDir, ok := history[acct.Name()]; ok {
		if oldDir != curDir {
			newDir = oldDir
			dirFound = true
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

	acct.Directories().Select(newDir)

	acct.Worker().PostAction(&types.RemoveDirectory{
		Directory: curDir,
		Quiet:     r.Force,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			app.PushStatus("Directory removed.", 10*time.Second)
		case *types.Error:
			app.PushError(msg.Error.Error())
		case *types.Unsupported:
			app.PushError(":rmdir is not supported by the backend.")
		}
	})

	return nil
}
