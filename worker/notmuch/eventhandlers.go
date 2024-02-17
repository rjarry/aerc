//go:build notmuch
// +build notmuch

package notmuch

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (w *worker) handleNotmuchEvent() error {
	err := w.db.Connect()
	if err != nil {
		return err
	}
	defer w.db.Close()
	err = w.updateDirCounts()
	if err != nil {
		return err
	}
	err = w.updateChangedMessages()
	if err != nil {
		return err
	}
	w.emitLabelList()
	return nil
}

func (w *worker) updateDirCounts() error {
	if w.store != nil {
		folders, err := w.store.FolderMap()
		if err != nil {
			w.w.Errorf("failed listing directories: %v", err)
			return err
		}
		for name := range folders {
			folder := filepath.Join(w.maildirAccountPath, name)
			query := fmt.Sprintf("folder:%s", strconv.Quote(folder))
			w.w.PostMessage(&types.DirectoryInfo{
				Info:    w.getDirectoryInfo(name, query),
				Refetch: w.query == query,
			}, nil)
		}
	}

	for name, query := range w.nameQueryMap {
		w.w.PostMessage(&types.DirectoryInfo{
			Info:    w.getDirectoryInfo(name, query),
			Refetch: w.query == query,
		}, nil)
	}

	for name, query := range w.dynamicNameQueryMap {
		w.w.PostMessage(&types.DirectoryInfo{
			Info:    w.getDirectoryInfo(name, query),
			Refetch: w.query == query,
		}, nil)
	}

	return nil
}

func (w *worker) updateChangedMessages() error {
	newState := w.db.State()
	if newState == w.state {
		return nil
	}
	w.w.Logger.Debugf("State change: %d to %d", w.state, newState)
	query := fmt.Sprintf("%s lastmod:%d..%d", w.query, w.state, newState)
	uids, err := w.uidsFromQuery(context.TODO(), query)
	if err != nil {
		return fmt.Errorf("Couldn't get updates messages: %w", err)
	}
	for _, uid := range uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			log.Errorf("%s", err)
			continue
		}
		err = w.emitMessageInfo(m, nil)
		if err != nil {
			log.Errorf("%s", err)
		}
	}
	w.state = newState
	return nil
}
