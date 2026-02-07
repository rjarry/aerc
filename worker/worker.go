package worker

import (
	"net/url"
	"strings"

	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

// Guesses the appropriate worker type based on the given source string
func NewWorker(source string, name string, messages chan types.WorkerMessage) (*types.Worker, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	worker := types.NewWorker(name, messages)
	scheme := u.Scheme
	if strings.ContainsRune(scheme, '+') {
		scheme = scheme[:strings.IndexRune(scheme, '+')]
	}
	backend, err := handlers.GetHandlerForScheme(scheme, worker)
	if err != nil {
		return nil, err
	}
	worker.Backend = backend
	return worker, nil
}
