package commands

import (
	"errors"
	"os"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("pwd", PrintWorkDirectory)
}

func PrintWorkDirectory(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: pwd")
	}
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	aerc.PushStatus(pwd, 10*time.Second)
	return nil
}
