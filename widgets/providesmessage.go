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
}

type ProvidesMessage interface {
	ui.Drawable
	Store() *lib.MessageStore
	SelectedAccount() *AccountView
	SelectedMessage() (*models.MessageInfo, error)
	SelectedMessagePart() *PartInfo
}

type ProvidesMessages interface {
	ui.Drawable
	Store() *lib.MessageStore
	SelectedAccount() *AccountView
	SelectedMessage() (*models.MessageInfo, error)
	MarkedMessages() ([]*models.MessageInfo, error)
}
