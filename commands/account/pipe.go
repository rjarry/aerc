package account

import (
	"errors"
	"io"
	"os/exec"
	"time"

	"git.sr.ht/~sircmpwn/aerc2/widgets"

	"github.com/gdamore/tcell"
)

func init() {
	register("pipe", Pipe)
}

func Pipe(aerc *widgets.Aerc, args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: :pipe <cmd> [args...]")
	}
	acct := aerc.SelectedAccount()
	store := acct.Messages().Store()
	msg := acct.Messages().Selected()
	store.FetchBodies([]uint32{msg.Uid}, func(reader io.Reader) {
		cmd := exec.Command(args[1], args[2:]...)
		pipe, err := cmd.StdinPipe()
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
			return
		}
		term, err := widgets.NewTerminal(cmd)
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
			return
		}
		name := msg.Envelope.Subject
		if len(name) > 12 {
			name = name[:12]
		}
		aerc.NewTab(term, args[1] + " <" + name)
		term.OnClose = func(err error) {
			if err != nil {
				aerc.PushStatus(" "+err.Error(), 10*time.Second).
					Color(tcell.ColorDefault, tcell.ColorRed)
			} else {
				// TODO: Tab-specific status stacks
				aerc.PushStatus("Process complete, press any key to close.",
					10*time.Second)
			}
		}
		term.OnStart = func() {
			go func() {
				_, err := io.Copy(pipe, reader)
				if err != nil {
					aerc.PushStatus(" "+err.Error(), 10*time.Second).
						Color(tcell.ColorDefault, tcell.ColorRed)
				}
				pipe.Close()
			}()
		}
	})
	return nil
}
