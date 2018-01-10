package worker

import (
	"git.sr.ht/~sircmpwn/aerc2/worker/imap"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type Worker interface {
	GetMessage() types.WorkerMessage
	PostAction(types.WorkerMessage)
	Run()
}

// Guesses the appropriate worker type based on the given source string
func NewWorker(source string) Worker {
	// TODO: Do this properly
	return imap.NewIMAPWorker()
}
