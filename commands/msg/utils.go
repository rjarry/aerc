package msg

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
)

type helper struct {
	msgProvider app.ProvidesMessages
	statusInfo  func(string)
}

func newHelper() *helper {
	msgProvider, ok := app.SelectedTabContent().(app.ProvidesMessages)
	if !ok {
		msgProvider = app.SelectedAccount()
	}
	return &helper{
		msgProvider: msgProvider,
		statusInfo: func(s string) {
			app.PushStatus(s, 10*time.Second)
		},
	}
}

func (h *helper) markedOrSelectedUids() ([]uint32, error) {
	return commands.MarkedOrSelected(h.msgProvider)
}

func (h *helper) store() (*lib.MessageStore, error) {
	store := h.msgProvider.Store()
	if store == nil {
		return nil, errors.New("Cannot perform action. Messages still loading")
	}
	return store, nil
}

func (h *helper) account() (*app.AccountView, error) {
	acct := h.msgProvider.SelectedAccount()
	if acct == nil {
		return nil, errors.New("No account selected")
	}
	return acct, nil
}

func (h *helper) messages() ([]*models.MessageInfo, error) {
	uid, err := commands.MarkedOrSelected(h.msgProvider)
	if err != nil {
		return nil, err
	}
	store, err := h.store()
	if err != nil {
		return nil, err
	}
	return commands.MsgInfoFromUids(store, uid, h.statusInfo)
}
