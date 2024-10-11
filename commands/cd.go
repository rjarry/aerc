package commands

import (
	"errors"
	"os"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

var previousDir string

type ChangeDirectory struct {
	Target string `opt:"directory" default:"~" complete:"CompleteTarget"`
}

func init() {
	Register(ChangeDirectory{})
}

func (ChangeDirectory) Description() string {
	return "Change aerc's current working directory."
}

func (ChangeDirectory) Context() CommandContext {
	return GLOBAL
}

func (ChangeDirectory) Aliases() []string {
	return []string{"cd"}
}

func (*ChangeDirectory) CompleteTarget(arg string) []string {
	return CompletePath(arg, true)
}

func (cd ChangeDirectory) Execute(args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if cd.Target == "-" {
		if previousDir == "" {
			return errors.New("No previous folder to return to")
		} else {
			cd.Target = previousDir
		}
	}
	target := xdg.ExpandHome(cd.Target)
	if err := os.Chdir(target); err == nil {
		previousDir = cwd
		app.UpdateStatus()
	}
	return err
}
