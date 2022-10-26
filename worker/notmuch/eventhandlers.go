//go:build notmuch
// +build notmuch

package notmuch

import (
	"fmt"
	"strconv"

	"git.sr.ht/~rjarry/aerc/logging"
)

func (w *worker) handleNotmuchEvent(et eventType) error {
	switch ev := et.(type) {
	case *updateDirCounts:
		return w.handleUpdateDirCounts(ev)
	default:
		return errUnsupported
	}
}

func (w *worker) handleUpdateDirCounts(ev eventType) error {
	if w.store != nil {
		folders, err := w.store.FolderMap()
		if err != nil {
			logging.Errorf("failed listing directories: %v", err)
			return err
		}
		for name := range folders {
			query := fmt.Sprintf("folder:%s", strconv.Quote(name))
			info, err := w.buildDirInfo(name, query, true)
			if err != nil {
				logging.Errorf("could not gather DirectoryInfo: %v", err)
				continue
			}
			w.w.PostMessage(info, nil)
		}
	}

	for name, query := range w.nameQueryMap {
		info, err := w.buildDirInfo(name, query, true)
		if err != nil {
			logging.Errorf("could not gather DirectoryInfo: %v", err)
			continue
		}
		w.w.PostMessage(info, nil)
	}
	return nil
}
