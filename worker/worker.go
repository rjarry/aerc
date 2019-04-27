package worker

import (
	"git.sr.ht/~sircmpwn/aerc2/worker/imap"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"

	"fmt"
	"log"
	"net/url"
)

// Guesses the appropriate worker type based on the given source string
func NewWorker(source string, logger *log.Logger) (*types.Worker, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	worker := types.NewWorker(logger)
	switch u.Scheme {
	case "imap":
		fallthrough
	case "imaps":
		worker.Backend = imap.NewIMAPWorker(worker)
	default:
		return nil, fmt.Errorf("Unknown backend %s", u.Scheme)
	}
	return worker, nil
}
