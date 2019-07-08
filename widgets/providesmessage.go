package widgets

import (
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/models"
)

type PartInfo struct {
	Index []int
	Msg   *models.MessageInfo
	Part  *models.BodyStructure
	Store *lib.MessageStore
}

type ProvidesMessage interface {
	ui.Drawable
	Store() *lib.MessageStore
	SelectedAccount() *AccountView
	SelectedMessage() *models.MessageInfo
	SelectedMessagePart() *PartInfo
}
