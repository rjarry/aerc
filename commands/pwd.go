package commands

import (
	"os"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
)

type PrintWorkDir struct{}

func init() {
	register(PrintWorkDir{})
}

func (PrintWorkDir) Aliases() []string {
	return []string{"pwd"}
}

func (PrintWorkDir) Complete(args []string) []string {
	return nil
}

func (PrintWorkDir) Execute(args []string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	app.PushStatus(pwd, 10*time.Second)
	return nil
}
