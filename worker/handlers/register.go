package handlers

import (
	"fmt"
	"runtime"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

type FactoryFunc func(*types.Worker) (types.Backend, error)

var workerFactories map[string]FactoryFunc = make(map[string]FactoryFunc)

func RegisterWorkerFactory(scheme string, factory FactoryFunc) {
	workerFactories[scheme] = factory
}

func GetHandlerForScheme(scheme string, worker *types.Worker) (types.Backend, error) {
	factory, ok := workerFactories[scheme]
	if !ok {
		return nil, fmt.Errorf("Unknown backend %s", scheme)
	}
	backend, err := factory(worker)
	if err != nil {
		return nil, err
	}
	return backend, nil
}

type WatcherFactoryFunc func() (types.FSWatcher, error)

var watcherFactories map[string]WatcherFactoryFunc = make(map[string]WatcherFactoryFunc)

func RegisterWatcherFactory(os string, fn WatcherFactoryFunc) {
	watcherFactories[os] = fn
}

func NewWatcher() (types.FSWatcher, error) {
	if fn, ok := watcherFactories[runtime.GOOS]; ok {
		return fn()
	}
	return nil, fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
}
