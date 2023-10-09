package app

import (
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
)

type PartInfo struct {
	Index []int
	Msg   *models.MessageInfo
	Part  *models.BodyStructure
	Links []string
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
	MarkedMessages() ([]uint32, error)
}
