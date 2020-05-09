package msg

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type helper struct {
	msgProvider widgets.ProvidesMessages
}

func newHelper(aerc *widgets.Aerc) *helper {
	return &helper{aerc.SelectedTab().(widgets.ProvidesMessages)}
}

func (h *helper) markedOrSelectedUids() ([]uint32, error) {
	msgs, err := commands.MarkedOrSelected(h.msgProvider)
	if err != nil {
		return nil, err
	}
	uids := commands.UidsFromMessageInfos(msgs)
	return uids, nil
}

func (h *helper) store() (*lib.MessageStore, error) {
	store := h.msgProvider.Store()
	if store == nil {
		return nil, errors.New("Cannot perform action. Messages still loading")
	}
	return store, nil
}

func (h *helper) account() (*widgets.AccountView, error) {
	acct := h.msgProvider.SelectedAccount()
	if acct == nil {
		return nil, errors.New("No account selected")
	}
	return acct, nil
}

func (h *helper) messages() ([]*models.MessageInfo, error) {
	return commands.MarkedOrSelected(h.msgProvider)
}
