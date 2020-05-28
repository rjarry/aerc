package commands

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type ExecCmd struct{}

func init() {
	register(ExecCmd{})
}

func (ExecCmd) Aliases() []string {
	return []string{"exec"}
}

func (ExecCmd) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (ExecCmd) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: exec [cmd...]")
	}
	cmd := exec.Command(args[1], args[2:]...)
	go func() {
		err := cmd.Run()
		if err != nil {
			aerc.PushError(" "+err.Error(), 10*time.Second)
		} else {
			if cmd.ProcessState.ExitCode() != 0 {
				aerc.PushError(fmt.Sprintf(
					"%s: completed with status %d", args[0],
					cmd.ProcessState.ExitCode()), 10*time.Second)
			} else {
				aerc.PushStatus(fmt.Sprintf(
					"%s: completed with status %d", args[0],
					cmd.ProcessState.ExitCode()), 10*time.Second)
			}
		}
	}()
	return nil
}
