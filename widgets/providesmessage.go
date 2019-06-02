package widgets

import (
	"git.sr.ht/~sircmpwn/aerc/worker/types"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
)

type ProvidesMessage interface {
	ui.Drawable
	Store()           *lib.MessageStore
	SelectedMessage() *types.MessageInfo
	SelectedAccount() *AccountView
}
