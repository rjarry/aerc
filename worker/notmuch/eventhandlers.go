//go:build notmuch
// +build notmuch

package notmuch

import "git.sr.ht/~rjarry/aerc/logging"

func (w *worker) handleNotmuchEvent(et eventType) error {
	switch ev := et.(type) {
	case *updateDirCounts:
		return w.handleUpdateDirCounts(ev)
	default:
		return errUnsupported
	}
}

func (w *worker) handleUpdateDirCounts(ev eventType) error {
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
