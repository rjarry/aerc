package watchers

import (
	"fmt"
	"runtime"
)

// FSWatcher is a file system watcher
type FSWatcher interface {
	Configure(string) error
	Events() chan *FSEvent
	// Adds a directory or file to the watcher
	Add(string) error
	// Removes a directory or file from the watcher
	Remove(string) error
}

type FSOperation int

const (
	FSCreate FSOperation = iota
	FSRemove
	FSRename
)

type FSEvent struct {
	Operation FSOperation
	Path      string
}

type WatcherFactoryFunc func() (FSWatcher, error)

var watcherFactory WatcherFactoryFunc

func RegisterWatcherFactory(fn WatcherFactoryFunc) {
	watcherFactory = fn
}

func NewWatcher() (FSWatcher, error) {
	if watcherFactory == nil {
		return nil, fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}
	return watcherFactory()
}
