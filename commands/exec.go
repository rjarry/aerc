package commands

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"

	"github.com/gdamore/tcell"
)

type ExecCmd struct{}

func init() {
	register(ExecCmd{})
}

func (_ ExecCmd) Aliases() []string {
	return []string{"exec"}
}

func (_ ExecCmd) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ ExecCmd) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: exec [cmd...]")
	}
	cmd := exec.Command(args[1], args[2:]...)
	go func() {
		err := cmd.Run()
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		} else {
			aerc.PushStatus(fmt.Sprintf(
				"%s: complete", args[0]), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorDefault)
		}
	}()
	return nil
}
