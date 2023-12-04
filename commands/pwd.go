package commands

import (
	"os"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
)

type PrintWorkDir struct{}

func init() {
	Register(PrintWorkDir{})
}

func (PrintWorkDir) Context() CommandContext {
	return GLOBAL
}

func (PrintWorkDir) Aliases() []string {
	return []string{"pwd"}
}

func (PrintWorkDir) Execute(args []string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	app.PushStatus(pwd, 10*time.Second)
	return nil
}
