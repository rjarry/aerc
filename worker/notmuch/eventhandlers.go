//go:build notmuch
// +build notmuch

package notmuch

import (
	"fmt"
	"strconv"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (w *worker) handleNotmuchEvent(et eventType) error {
	switch et.(type) {
	case *updateDirCounts:
		return w.handleUpdateDirCounts()
	default:
		return errUnsupported
	}
}

func (w *worker) handleUpdateDirCounts() error {
	if w.store != nil {
		folders, err := w.store.FolderMap()
		if err != nil {
			log.Errorf("failed listing directories: %v", err)
			return err
		}
		for name := range folders {
			query := fmt.Sprintf("folder:%s", strconv.Quote(name))
			w.w.PostMessage(&types.DirectoryInfo{
				Info:     w.getDirectoryInfo(name, query),
				SkipSort: true,
			}, nil)
		}
	}

	for name, query := range w.nameQueryMap {
		w.w.PostMessage(&types.DirectoryInfo{
			Info:     w.getDirectoryInfo(name, query),
			SkipSort: true,
		}, nil)
	}
	return nil
}
