package commands

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/log"
)

type ExecCmd struct {
	Args []string `opt:"..."`
}

func init() {
	Register(ExecCmd{})
}

func (ExecCmd) Context() CommandContext {
	return GLOBAL
}

func (ExecCmd) Aliases() []string {
	return []string{"exec"}
}

func (e ExecCmd) Execute(args []string) error {
	cmd := exec.Command(e.Args[0], e.Args[1:]...)
	env := os.Environ()

	switch view := app.SelectedTabContent().(type) {
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
			app.PushError(err.Error())
		} else {
			if cmd.ProcessState.ExitCode() != 0 {
				app.PushError(fmt.Sprintf(
					"%s: completed with status %d", args[0],
					cmd.ProcessState.ExitCode()))
			} else {
				app.PushStatus(fmt.Sprintf(
					"%s: completed with status %d", args[0],
					cmd.ProcessState.ExitCode()), 10*time.Second)
			}
		}
	}()
	return nil
}
