package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

type Eml struct {
	Path string `opt:"path" required:"false" complete:"CompletePath"`
}

func init() {
	Register(Eml{})
}

func (Eml) Context() CommandContext {
	return GLOBAL
}

func (Eml) Aliases() []string {
	return []string{"eml", "preview"}
}

func (*Eml) CompletePath(arg string) []string {
	return CompletePath(arg, false)
}

func (e Eml) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return fmt.Errorf("no account selected")
	}

	showEml := func(r io.Reader) {
		data, err := io.ReadAll(r)
		if err != nil {
			app.PushError(err.Error())
			return
		}
		lib.NewEmlMessageView(data, app.CryptoProvider(), app.DecryptKeys,
			func(view lib.MessageView, err error) {
				if err != nil {
					app.PushError(err.Error())
					return
				}
				msgView, err := app.NewMessageViewer(acct, view)
				if err != nil {
					app.PushError(err.Error())
					return
				}
				app.NewTab(msgView,
					view.MessageInfo().Envelope.Subject)
			})
	}

	if e.Path == "" {
		switch tab := app.SelectedTabContent().(type) {
		case *app.MessageViewer:
			part := tab.SelectedMessagePart()
			tab.MessageView().FetchBodyPart(part.Index, showEml)
		case *app.Composer:
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
		f, err := os.Open(xdg.ExpandHome(e.Path))
		if err != nil {
			return err
		}
		defer f.Close()
		showEml(f)
	}
	return nil
}
