package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/widgets"
)

type Eml struct{}

func init() {
	register(Eml{})
}

func (Eml) Aliases() []string {
	return []string{"eml", "preview"}
}

func (Eml) Complete(aerc *widgets.Aerc, args []string) []string {
	return CompletePath(strings.Join(args, " "))
}

func (Eml) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return fmt.Errorf("no account selected")
	}

	showEml := func(r io.Reader) {
		data, err := io.ReadAll(r)
		if err != nil {
			aerc.PushError(err.Error())
			return
		}
		lib.NewEmlMessageView(data, aerc.Crypto, aerc.DecryptKeys,
			func(view lib.MessageView, err error) {
				if err != nil {
					aerc.PushError(err.Error())
					return
				}
				msgView := widgets.NewMessageViewer(acct, view)
				aerc.NewTab(msgView,
					view.MessageInfo().Envelope.Subject)
			})
	}

	if len(args) == 1 {
		switch tab := aerc.SelectedTabContent().(type) {
		case *widgets.MessageViewer:
			part := tab.SelectedMessagePart()
			tab.MessageView().FetchBodyPart(part.Index, showEml)
		case *widgets.Composer:
			var buf bytes.Buffer
			h, err := tab.PrepareHeader()
			if err != nil {
				return err
			}
			if err := tab.WriteMessage(h, &buf); err != nil {
				return err
			}
			showEml(&buf)
		default:
			return fmt.Errorf("unsupported operation")
		}
	} else {
		path := strings.Join(args[1:], " ")
		if _, err := os.Stat(path); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		showEml(f)
	}
	return nil
}
