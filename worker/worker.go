package worker

import (
	"git.sr.ht/~sircmpwn/aerc/worker/imap"
	"git.sr.ht/~sircmpwn/aerc/worker/maildir"
	"git.sr.ht/~sircmpwn/aerc/worker/types"

	"fmt"
	"log"
	"net/url"
	"strings"
)

// Guesses the appropriate worker type based on the given source string
func NewWorker(source string, logger *log.Logger) (*types.Worker, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	worker := types.NewWorker(logger)
	scheme := u.Scheme
	if strings.ContainsRune(scheme, '+') {
		scheme = scheme[:strings.IndexRune(scheme, '+')]
		fmt.Println(scheme)
	}
	switch scheme {
	case "imap":
		fallthrough
	case "imaps":
		worker.Backend = imap.NewIMAPWorker(worker)
	case "maildir":
		worker.Backend = maildir.NewWorker(worker)
	default:
		return nil, fmt.Errorf("Unknown backend %s", u.Scheme)
	}
	return worker, nil
}
