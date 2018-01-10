package worker

import (
	"git.sr.ht/~sircmpwn/aerc2/worker/imap"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"

	"fmt"
	"net/url"
)

type Worker interface {
	GetMessage() types.WorkerMessage
	PostAction(types.WorkerMessage)
	Run()
}

// Guesses the appropriate worker type based on the given source string
func NewWorker(source string) (Worker, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "imap":
	case "imaps":
		return imap.NewIMAPWorker(), nil
	}
	return nil, fmt.Errorf("Unknown backend %s", u.Scheme)
}
