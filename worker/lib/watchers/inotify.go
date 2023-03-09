//go:build !darwin
// +build !darwin

package watchers

import (
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/fsnotify/fsnotify"
)

func init() {
	handlers.RegisterWatcherFactory(newInotifyWatcher)
}

type inotifyWatcher struct {
	w  *fsnotify.Watcher
	ch chan *types.FSEvent
}

func newInotifyWatcher() (types.FSWatcher, error) {
	watcher := &inotifyWatcher{
		ch: make(chan *types.FSEvent),
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
			w.ch <- &types.FSEvent{
				Operation: types.FSCreate,
				Path:      ev.Name,
			}
		case fsnotify.Remove:
			w.ch <- &types.FSEvent{
				Operation: types.FSRemove,
				Path:      ev.Name,
			}
		case fsnotify.Rename:
			w.ch <- &types.FSEvent{
				Operation: types.FSRename,
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

func (w *inotifyWatcher) Events() chan *types.FSEvent {
	return w.ch
}

func (w *inotifyWatcher) Add(p string) error {
	return w.w.Add(p)
}

func (w *inotifyWatcher) Remove(p string) error {
	return w.w.Remove(p)
}
