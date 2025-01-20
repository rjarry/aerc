package marker

import "git.sr.ht/~rjarry/aerc/models"

// Marker provides the interface for the marking behavior of messages
type Marker interface {
	Mark(models.UID)
	Unmark(models.UID)
	ToggleMark(models.UID)
	Remark()
	Marked() []models.UID
	IsMarked(models.UID) bool
	IsVisualMark() bool
	ToggleVisualMark(bool)
	UpdateVisualMark()
	ClearVisualMark()
}

// UIDProvider provides the underlying uids and the selected message index
type UIDProvider interface {
	Uids() []models.UID
	SelectedIndex() int
}

type controller struct {
	uidProvider    UIDProvider
	marked         map[models.UID]struct{}
	lastMarked     map[models.UID]struct{}
	visualStartUID models.UID
	visualMarkMode bool
	visualBase     map[models.UID]struct{}
}

// New returns a new Marker
func New(up UIDProvider) Marker {
	return &controller{
		uidProvider: up,
		marked:      make(map[models.UID]struct{}),
		lastMarked:  make(map[models.UID]struct{}),
	}
}

// Mark marks the uid as marked
func (mc *controller) Mark(uid models.UID) {
	if mc.visualMarkMode {
		// visual mode has override, bogus input from user
		return
	}
	mc.marked[uid] = struct{}{}
}

// Unmark unmarks the uid
func (mc *controller) Unmark(uid models.UID) {
	if mc.visualMarkMode {
		// user probably wanted to clear the visual marking
		mc.ClearVisualMark()
		return
	}
	delete(mc.marked, uid)
}

// Remark restores the previous marks
func (mc *controller) Remark() {
	mc.marked = mc.lastMarked
}

// ToggleMark toggles the marked state for the given uid
func (mc *controller) ToggleMark(uid models.UID) {
	if mc.visualMarkMode {
		// visual mode has override, bogus input from user
		return
	}
	if mc.IsMarked(uid) {
		mc.Unmark(uid)
	} else {
		mc.Mark(uid)
	}
}

// resetMark removes the marking from all messages
func (mc *controller) resetMark() {
	mc.lastMarked = mc.marked
	mc.marked = make(map[models.UID]struct{})
}

// removeStaleUID removes uids that are no longer presents in the UIDProvider
func (mc *controller) removeStaleUID() {
	for mark := range mc.marked {
		present := false
		for _, uid := range mc.uidProvider.Uids() {
			if mark == uid {
				present = true
				break
			}
		}
		if !present {
			delete(mc.marked, mark)
		}
	}
}

// IsMarked checks whether the given uid has been marked
func (mc *controller) IsMarked(uid models.UID) bool {
	_, marked := mc.marked[uid]
	return marked
}

// Marked returns the uids of all marked messages
func (mc *controller) Marked() []models.UID {
	mc.removeStaleUID()
	marked := make([]models.UID, len(mc.marked))
	i := 0
	for uid := range mc.marked {
		marked[i] = uid
		i++
	}
	return marked
}

// IsVisualMark indicates whether visual marking mode is enabled.
func (mc *controller) IsVisualMark() bool {
	return mc.visualMarkMode
}

// ToggleVisualMark enters or leaves the visual marking mode
func (mc *controller) ToggleVisualMark(clear bool) {
	mc.visualMarkMode = !mc.visualMarkMode
	if mc.visualMarkMode {
		// just entered visual mode, reset whatever marking was already done
		if clear {
			mc.resetMark()
		}
		uids := mc.uidProvider.Uids()
		if idx := mc.uidProvider.SelectedIndex(); idx >= 0 && idx < len(uids) {
			mc.visualStartUID = uids[idx]
			mc.marked[mc.visualStartUID] = struct{}{}
			mc.visualBase = make(map[models.UID]struct{})
			for key, value := range mc.marked {
				mc.visualBase[key] = value
			}
		}
	}
}

// ClearVisualMark leaves the visual marking mode and resets any marking
func (mc *controller) ClearVisualMark() {
	mc.resetMark()
	mc.visualMarkMode = false
	mc.visualStartUID = ""
}

// UpdateVisualMark updates the index with the currently selected message
func (mc *controller) UpdateVisualMark() {
	if !mc.visualMarkMode {
		// nothing to do
		return
	}
	startIdx := mc.visualStartIdx()
	if startIdx < 0 {
		// something deleted the startuid, abort the marking process
		mc.ClearVisualMark()
		return
	}

	selectedIdx := mc.uidProvider.SelectedIndex()
	if selectedIdx < 0 {
		return
	}

	uids := mc.uidProvider.Uids()

	var visUids []models.UID
	if selectedIdx > startIdx {
		visUids = uids[startIdx : selectedIdx+1]
	} else {
		visUids = uids[selectedIdx : startIdx+1]
	}
	mc.marked = make(map[models.UID]struct{})
	for uid := range mc.visualBase {
		mc.marked[uid] = struct{}{}
	}
	for _, uid := range visUids {
		mc.marked[uid] = struct{}{}
	}
}

// returns the index of needle in haystack or -1 if not found
func (mc *controller) visualStartIdx() int {
	for idx, u := range mc.uidProvider.Uids() {
		if u == mc.visualStartUID {
			return idx
		}
	}
	return -1
}
