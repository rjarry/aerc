package commands

import (
	"io"
	"os/exec"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"github.com/gdamore/tcell"
)

// QuickTerm is an ephemeral terminal for running a single command and quiting.
func QuickTerm(aerc *widgets.Aerc, args []string, stdin io.Reader) (*widgets.Terminal, error) {
	cmd := exec.Command(args[0], args[1:]...)
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	term, err := widgets.NewTerminal(cmd)
	if err != nil {
		return nil, err
	}

	term.OnClose = func(err error) {
		if err != nil {
			aerc.PushError(" " + err.Error())
			// remove the tab on error, otherwise it gets stuck
			aerc.RemoveTab(term)
		} else {
			aerc.PushStatus("Process complete, press any key to close.",
				10*time.Second)
			term.OnEvent = func(event tcell.Event) bool {
				aerc.RemoveTab(term)
				return true
			}
		}
	}

	term.OnStart = func() {
		status := make(chan error, 1)

		go func() {
			_, err := io.Copy(pipe, stdin)
			defer pipe.Close()
			status <- err
		}()

		err := <-status
		if err != nil {
			aerc.PushError(" " + err.Error())
		}
	}

	return term, nil
}
