//go:build darwin

package watchers

import (
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/fsnotify/fsevents"
)

func init() {
	RegisterWatcherFactory(newDarwinWatcher)
}

type darwinWatcher struct {
	ch        chan *FSEvent
	w         *fsevents.EventStream
	watcherCh chan []fsevents.Event
}

func newDarwinWatcher() (FSWatcher, error) {
	watcher := &darwinWatcher{
		watcherCh: make(chan []fsevents.Event),
		ch:        make(chan *FSEvent),
		w: &fsevents.EventStream{
			Flags:   fsevents.WatchRoot,
			Latency: 500 * time.Millisecond,
		},
	}
	return watcher, nil
}

func (w *darwinWatcher) watch() {
	defer log.PanicHandler()
	for events := range w.w.Events {
		for _, ev := range events {
			// The Item* flags are only reported when the stream is
			// created with fsevents.FileEvents. Directory granularity
			// events coalesce into a single event with none of them
			// set, so anything else must be reported as a change.
			op := FSCreate
			switch {
			case ev.Flags&fsevents.ItemRenamed > 0:
				op = FSRename
			case ev.Flags&fsevents.ItemRemoved > 0:
				op = FSRemove
			}
			w.ch <- &FSEvent{
				Operation: op,
				Path:      ev.Path,
			}
		}
	}
}

func (w *darwinWatcher) Configure(root string) error {
	dev, err := fsevents.DeviceForPath(root)
	if err != nil {
		return err
	}
	w.w.Device = dev
	w.w.Paths = []string{root}
	start_err := w.w.Start()
	if start_err != nil {
		return start_err
	}
	go w.watch()
	return nil
}

func (w *darwinWatcher) Events() chan *FSEvent {
	return w.ch
}

func (w *darwinWatcher) Add(p string) error {
	return nil
}

func (w *darwinWatcher) Remove(p string) error {
	return nil
}
