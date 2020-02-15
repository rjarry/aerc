package notmuch

func (w *worker) handleNotmuchEvent(et eventType) error {
	switch ev := et.(type) {
	default:
		_ = ev
		return errUnsupported
	}
}
