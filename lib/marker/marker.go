package marker

import (
	"maps"
	"slices"
	"sync"

	"git.sr.ht/~rjarry/aerc/models"
)

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
	markedMtx      sync.RWMutex
	marked         map[models.UID]struct{}
	lastMarked     map[models.UID]struct{}
	visualStartUID models.UID
	visualMarkMode bool
	visualBaseMtx  sync.RWMutex
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
	mc.markedMtx.Lock()
	mc.marked[uid] = struct{}{}
	mc.markedMtx.Unlock()
}

// Unmark unmarks the uid
func (mc *controller) Unmark(uid models.UID) {
	if mc.visualMarkMode {
		// user probably wanted to clear the visual marking
		mc.ClearVisualMark()
		return
	}
	mc.markedMtx.Lock()
	delete(mc.marked, uid)
	mc.markedMtx.Unlock()
}

// Remark restores the previous marks
func (mc *controller) Remark() {
	mc.markedMtx.Lock()
	mc.marked = mc.lastMarked
	mc.markedMtx.Unlock()
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
	mc.markedMtx.Lock()
	mc.lastMarked = mc.marked
	mc.marked = make(map[models.UID]struct{})
	mc.markedMtx.Unlock()
}

// removeStaleUID removes uids that are no longer presents in the UIDProvider
func (mc *controller) removeStaleUID() {
	mc.markedMtx.Lock()
	defer mc.markedMtx.Unlock()
	for mark := range mc.marked {
		present := slices.Contains(mc.uidProvider.Uids(), mark)
		if !present {
			delete(mc.marked, mark)
		}
	}
}

// IsMarked checks whether the given uid has been marked
func (mc *controller) IsMarked(uid models.UID) bool {
	mc.markedMtx.RLock()
	_, marked := mc.marked[uid]
	mc.markedMtx.RUnlock()
	return marked
}

// Marked returns the uids of all marked messages
func (mc *controller) Marked() []models.UID {
	mc.removeStaleUID()
	marked := make([]models.UID, len(mc.marked))
	i := 0
	mc.markedMtx.RLock()
	defer mc.markedMtx.RUnlock()
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
			mc.markedMtx.Lock()
			mc.marked[mc.visualStartUID] = struct{}{}
			mc.markedMtx.Unlock()
			mc.visualBase = make(map[models.UID]struct{})
			mc.markedMtx.RLock()
			defer mc.markedMtx.RUnlock()
			mc.visualBaseMtx.Lock()
			defer mc.visualBaseMtx.Unlock()
			maps.Copy(mc.visualBase, mc.marked)
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
	mc.markedMtx.Lock()
	defer mc.markedMtx.Unlock()
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
