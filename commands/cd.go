package commands

import (
	"errors"
	"os"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

var previousDir string

type ChangeDirectory struct {
	Target string `opt:"directory" default:"~" complete:"CompleteTarget"`
}

func init() {
	register(ChangeDirectory{})
}

func (ChangeDirectory) Aliases() []string {
	return []string{"cd"}
}

func (*ChangeDirectory) CompleteTarget(arg string) []string {
	completions := CompletePath(arg)

	var dirs []string
	for _, c := range completions {
		// filter out non-directories
		if strings.HasSuffix(c, "/") {
			dirs = append(dirs, c)
		}
	}

	return dirs
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
