package marker_test

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/marker"
)

// mockUidProvider implements the UidProvider interface and mocks the message
// store for testing
type mockUidProvider struct {
	uids []uint32
	idx  int
}

func (mock *mockUidProvider) Uids() []uint32 {
	return mock.uids
}

func (mock *mockUidProvider) SelectedIndex() int {
	return mock.idx
}

func createMarker() (marker.Marker, *mockUidProvider) {
	uidProvider := &mockUidProvider{
		uids: []uint32{1, 2, 3, 4},
		idx:  1,
	}
	m := marker.New(uidProvider)
	return m, uidProvider
}

func TestMarker_MarkUnmark(t *testing.T) {
	m, _ := createMarker()
	uid := uint32(4)

	m.Mark(uid)
	if !m.IsMarked(uid) {
		t.Errorf("Marking failed")
	}

	m.Unmark(uid)
	if m.IsMarked(uid) {
		t.Errorf("Unmarking failed")
	}
}

func TestMarker_ToggleMark(t *testing.T) {
	m, _ := createMarker()
	uid := uint32(4)

	if m.IsMarked(uid) {
		t.Errorf("ToggleMark: uid should not be marked")
	}

	m.ToggleMark(uid)
	if !m.IsMarked(uid) {
		t.Errorf("ToggleMark: uid should be marked")
	}

	m.ToggleMark(uid)
	if m.IsMarked(uid) {
		t.Errorf("ToggleMark: uid should not be marked")
	}
}

func TestMarker_Marked(t *testing.T) {
	m, _ := createMarker()
	expected := map[uint32]struct{}{
		uint32(1): {},
		uint32(4): {},
	}
	for uid := range expected {
		m.Mark(uid)
	}

	got := m.Marked()
	if len(expected) != len(got) {
		t.Errorf("Marked: expected len of %d but got %d", len(expected), len(got))
	}

	for _, uid := range got {
		if _, ok := expected[uid]; !ok {
			t.Errorf("Marked: received uid %d as marked but it should not be", uid)
		}
	}
}

func TestMarker_VisualMode(t *testing.T) {
	m, up := createMarker()

	// activate visual mode
	m.ToggleVisualMark()

	// marking should now fail silently because we're in visual mode
	m.Mark(1)
	if m.IsMarked(1) {
		t.Errorf("marking in visual mode should not work")
	}

	// move selection index to last item
	up.idx = len(up.uids) - 1
	m.UpdateVisualMark()
	expectedMarked := []uint32{2, 3, 4}

	for _, uidMarked := range expectedMarked {
		if !m.IsMarked(uidMarked) {
			t.Logf("expected: %#v, got: %#v", expectedMarked, m.Marked())
			t.Errorf("updatevisual: uid %v should be marked in visual mode", uidMarked)
		}
	}

	// clear all
	m.ClearVisualMark()
	if len(m.Marked()) > 0 {
		t.Errorf("no uids should be marked after clearing visual mark")
	}

	// test remark
	m.Remark()
	for _, uidMarked := range expectedMarked {
		if !m.IsMarked(uidMarked) {
			t.Errorf("remark: uid %v should be marked in visual mode", uidMarked)
		}
	}
}

func TestMarker_MarkOutOfBound(t *testing.T) {
	m, _ := createMarker()

	outOfBoundUid := uint32(100)

	m.Mark(outOfBoundUid)
	for _, markedUid := range m.Marked() {
		if markedUid == outOfBoundUid {
			t.Errorf("out-of-bound uid should not be marked")
		}
	}
}
