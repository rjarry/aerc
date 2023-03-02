package types

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
