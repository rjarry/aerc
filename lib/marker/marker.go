package marker

// Marker provides the interface for the marking behavior of messages
type Marker interface {
	Mark(uint32)
	Unmark(uint32)
	ToggleMark(uint32)
	Remark()
	Marked() []uint32
	IsMarked(uint32) bool
	ToggleVisualMark(bool)
	UpdateVisualMark()
	ClearVisualMark()
}

// UIDProvider provides the underlying uids and the selected message index
type UIDProvider interface {
	Uids() []uint32
	SelectedIndex() int
}

type controller struct {
	uidProvider    UIDProvider
	marked         map[uint32]struct{}
	lastMarked     map[uint32]struct{}
	visualStartUID uint32
	visualMarkMode bool
	visualBase     map[uint32]struct{}
}

// New returns a new Marker
func New(up UIDProvider) Marker {
	return &controller{
		uidProvider: up,
		marked:      make(map[uint32]struct{}),
		lastMarked:  make(map[uint32]struct{}),
	}
}

// Mark markes the uid as marked
func (mc *controller) Mark(uid uint32) {
	if mc.visualMarkMode {
		// visual mode has override, bogus input from user
		return
	}
	mc.marked[uid] = struct{}{}
}

// Unmark unmarks the uid
func (mc *controller) Unmark(uid uint32) {
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
func (mc *controller) ToggleMark(uid uint32) {
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
	mc.marked = make(map[uint32]struct{})
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
func (mc *controller) IsMarked(uid uint32) bool {
	_, marked := mc.marked[uid]
	return marked
}

// Marked returns the uids of all marked messages
func (mc *controller) Marked() []uint32 {
	mc.removeStaleUID()
	marked := make([]uint32, len(mc.marked))
	i := 0
	for uid := range mc.marked {
		marked[i] = uid
		i++
	}
	return marked
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
			mc.visualBase = make(map[uint32]struct{})
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
	mc.visualStartUID = 0
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

	var visUids []uint32
	if selectedIdx > startIdx {
		visUids = uids[startIdx : selectedIdx+1]
	} else {
		visUids = uids[selectedIdx : startIdx+1]
	}
	mc.marked = make(map[uint32]struct{})
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
