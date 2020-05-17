package msg

import (
	"errors"
	"strings"

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
	return commands.MarkedOrSelected(h.msgProvider)
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
	uid, err := commands.MarkedOrSelected(h.msgProvider)
	if err != nil {
		return nil, err
	}
	store, err := h.store()
	if err != nil {
		return nil, err
	}
	return commands.MsgInfoFromUids(store, uid)
}

func findPlaintext(bs *models.BodyStructure, path []int) []int {
	for i, part := range bs.Parts {
		cur := append(path, i+1)
		if strings.ToLower(part.MIMEType) == "text" &&
			strings.ToLower(part.MIMESubType) == "plain" {
			return cur
		}
		if strings.ToLower(part.MIMEType) == "multipart" {
			if path := findPlaintext(part, cur); path != nil {
				return path
			}
		}
	}
	return nil
}

func findFirstNonMultipart(bs *models.BodyStructure, path []int) []int {
	for i, part := range bs.Parts {
		cur := append(path, i+1)
		mimetype := strings.ToLower(part.MIMEType)
		if mimetype != "multipart" {
			return path
		} else if mimetype == "multipart" {
			if path := findPlaintext(part, cur); path != nil {
				return path
			}
		}
	}
	return nil
}
