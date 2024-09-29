package account

import (
	"bytes"
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/models"
)

type ViewMessage struct {
	Peek       bool `opt:"-p"`
	Background bool `opt:"-b"`
}

func init() {
	commands.Register(ViewMessage{})
}

func (ViewMessage) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (ViewMessage) Aliases() []string {
	return []string{"view-message", "view"}
}

func (v ViewMessage) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if acct.Messages().Empty() {
		return nil
	}
	store := acct.Messages().Store()
	msg := acct.Messages().Selected()
	if msg == nil {
		return nil
	}
	_, deleted := store.Deleted[msg.Uid]
	if deleted {
		return nil
	}
	if msg.Error != nil {
		app.PushError(msg.Error.Error())
		return nil
	}
	lib.NewMessageStoreView(
		msg,
		!v.Peek && acct.UiConfig().AutoMarkRead,
		store,
		app.CryptoProvider(),
		app.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				app.PushError(err.Error())
				return
			}
			viewer, err := app.NewMessageViewer(acct, view)
			if err != nil {
				app.PushError(err.Error())
				return
			}
			data := state.NewDataSetter()
			data.SetAccount(acct.AccountConfig())
			data.SetFolder(acct.Directories().SelectedDirectory())
			data.SetHeaders(msg.RFC822Headers, &models.OriginalMail{})
			var buf bytes.Buffer
			err = templates.Render(acct.UiConfig().TabTitleViewer, &buf,
				data.Data())
			if err != nil {
				acct.PushError(err)
				return
			}
			if v.Background {
				app.NewBackgroundTab(viewer, buf.String())
			} else {
				app.NewTab(viewer, buf.String())
			}
		})
	return nil
}
