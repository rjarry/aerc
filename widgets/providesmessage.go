package widgets

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type PartInfo struct {
	Index []int
	Msg   *types.MessageInfo
	Part  *imap.BodyStructure
	Store *lib.MessageStore
}

type ProvidesMessage interface {
	ui.Drawable
	Store() *lib.MessageStore
	SelectedAccount() *AccountView
	SelectedMessage() *types.MessageInfo
	SelectedMessagePart() *PartInfo
}
