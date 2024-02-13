//go:build !darwin
// +build !darwin

package watchers

import (
	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/fsnotify/fsnotify"
)

func init() {
	RegisterWatcherFactory(newInotifyWatcher)
}

type inotifyWatcher struct {
	w  *fsnotify.Watcher
	ch chan *FSEvent
}

func newInotifyWatcher() (FSWatcher, error) {
	watcher := &inotifyWatcher{
		ch: make(chan *FSEvent),
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher.w = w

	go watcher.watch()
	return watcher, nil
}

func (w *inotifyWatcher) watch() {
	defer log.PanicHandler()
	for ev := range w.w.Events {
		// we only care about files being created, removed or renamed
		switch ev.Op {
		case fsnotify.Create:
			w.ch <- &FSEvent{
				Operation: FSCreate,
				Path:      ev.Name,
			}
		case fsnotify.Remove:
			w.ch <- &FSEvent{
				Operation: FSRemove,
				Path:      ev.Name,
			}
		case fsnotify.Rename:
			w.ch <- &FSEvent{
				Operation: FSRename,
				Path:      ev.Name,
			}
		default:
			continue
		}
	}
}

func (w *inotifyWatcher) Configure(root string) error {
	return w.w.Add(root)
}

func (w *inotifyWatcher) Events() chan *FSEvent {
	return w.ch
}

func (w *inotifyWatcher) Add(p string) error {
	return w.w.Add(p)
}

func (w *inotifyWatcher) Remove(p string) error {
	return w.w.Remove(p)
}
