package commands

import (
	"errors"
	"os"
	"strings"

	"git.sr.ht/~rjarry/aerc/widgets"
	"github.com/mitchellh/go-homedir"
)

var (
	previousDir string
)

type ChangeDirectory struct{}

func init() {
	register(ChangeDirectory{})
}

func (ChangeDirectory) Aliases() []string {
	return []string{"cd"}
}

func (ChangeDirectory) Complete(aerc *widgets.Aerc, args []string) []string {
	path := strings.Join(args, " ")
	completions := CompletePath(path)

	var dirs []string
	for _, c := range completions {
		// filter out non-directories
		if strings.HasSuffix(c, "/") {
			dirs = append(dirs, c)
		}
	}

	return dirs
}

func (ChangeDirectory) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 1 {
		return errors.New("Usage: cd [directory]")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	target := strings.Join(args[1:], " ")
	if target == "" {
		target = "~"
	} else if target == "-" {
		if previousDir == "" {
			return errors.New("No previous folder to return to")
		} else {
			target = previousDir
		}
	}
	target, err = homedir.Expand(target)
	if err != nil {
		return err
	}
	if err := os.Chdir(target); err == nil {
		previousDir = cwd
	}
	return err
}
