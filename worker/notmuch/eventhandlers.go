//go:build notmuch
// +build notmuch

package notmuch

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
		info, err := w.gatherDirectoryInfo(name, query)
		if err != nil {
			w.w.Logger.Printf("could not gather DirectoryInfo: %v\n", err)
			continue
		}
		w.w.PostMessage(info, nil)
	}
	return nil
}
