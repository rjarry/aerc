package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/log"
)

type ExecCmd struct{}

func init() {
	register(ExecCmd{})
}

func (ExecCmd) Aliases() []string {
	return []string{"exec"}
}

func (ExecCmd) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (ExecCmd) Execute(aerc *app.Aerc, args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: exec [cmd...]")
	}

	cmd := exec.Command(args[1], args[2:]...)
	env := os.Environ()

	switch view := aerc.SelectedTabContent().(type) {
	case *app.AccountView:
		env = append(env, fmt.Sprintf("account=%s", view.AccountConfig().Name))
		env = append(env, fmt.Sprintf("folder=%s", view.Directories().Selected()))
	case *app.MessageViewer:
		acct := view.SelectedAccount()
		env = append(env, fmt.Sprintf("account=%s", acct.AccountConfig().Name))
		env = append(env, fmt.Sprintf("folder=%s", acct.Directories().Selected()))
	}

	cmd.Env = env

	go func() {
		defer log.PanicHandler()

		err := cmd.Run()
		if err != nil {
			aerc.PushError(err.Error())
		} else {
			if cmd.ProcessState.ExitCode() != 0 {
				aerc.PushError(fmt.Sprintf(
					"%s: completed with status %d", args[0],
					cmd.ProcessState.ExitCode()))
			} else {
				aerc.PushStatus(fmt.Sprintf(
					"%s: completed with status %d", args[0],
					cmd.ProcessState.ExitCode()), 10*time.Second)
			}
		}
	}()
	return nil
}
