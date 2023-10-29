package types

import (
	"errors"
	"fmt"
	"sort"

	"git.sr.ht/~rjarry/aerc/log"
)

type Thread struct {
	Uid         uint32
	Parent      *Thread
	PrevSibling *Thread
	NextSibling *Thread
	FirstChild  *Thread

	Hidden  int  // if this flag is not zero the message isn't rendered in the UI
	Deleted bool // if this flag is set the message was deleted

	// Context indicates the message doesn't match the mailbox / query but
	// is displayed for context
	Context bool
}

// AddChild appends the child node at the end of the existing children of t.
func (t *Thread) AddChild(child *Thread) {
	t.InsertCmp(child, func(_, _ *Thread) bool { return true })
}

// OrderedInsert inserts the child node in ascending order among the existing
// children based on their respective UIDs.
func (t *Thread) OrderedInsert(child *Thread) {
	t.InsertCmp(child, func(child, iter *Thread) bool { return child.Uid > iter.Uid })
}

// InsertCmp inserts child as a child node into t in ascending order. The
// ascending order is determined by the bigger function that compares the child
// with the existing children. It should return true when the child is bigger
// than the other, and false otherwise.
func (t *Thread) InsertCmp(child *Thread, bigger func(*Thread, *Thread) bool) {
	if t.FirstChild == nil {
		t.FirstChild = child
	} else {
		start := &Thread{NextSibling: t.FirstChild}
		var iter *Thread
		for iter = start; iter.NextSibling != nil &&
			bigger(child, iter.NextSibling); iter = iter.NextSibling {
		}
		child.NextSibling = iter.NextSibling
		iter.NextSibling = child
		t.FirstChild = start.NextSibling
	}
	child.Parent = t
}

func (t *Thread) Walk(walkFn NewThreadWalkFn) error {
	err := newWalk(t, walkFn, 0, nil)
	if errors.Is(err, ErrSkipThread) {
		return nil
	}
	return err
}

// Root returns the root thread of the thread tree
func (t *Thread) Root() *Thread {
	if t == nil {
		return nil
	}
	var iter *Thread
	for iter = t; iter.Parent != nil; iter = iter.Parent {
	}
	return iter
}

// Uids returns all associated uids for the given thread and its children
func (t *Thread) Uids() []uint32 {
	if t == nil {
		return nil
	}
	uids := make([]uint32, 0)
	err := t.Walk(func(node *Thread, _ int, _ error) error {
		uids = append(uids, node.Uid)
		return nil
	})
	if err != nil {
		log.Errorf("walk to collect uids failed: %v", err)
	}
	return uids
}

func (t *Thread) String() string {
	if t == nil {
		return "<nil>"
	}
	parent := -1
	if t.Parent != nil {
		parent = int(t.Parent.Uid)
	}
	next := -1
	if t.NextSibling != nil {
		next = int(t.NextSibling.Uid)
	}
	child := -1
	if t.FirstChild != nil {
		child = int(t.FirstChild.Uid)
	}
	return fmt.Sprintf(
		"[%d] (parent:%v, next:%v, child:%v)",
		t.Uid, parent, next, child,
	)
}

func newWalk(node *Thread, walkFn NewThreadWalkFn, lvl int, ce error) error {
	if node == nil {
		return nil
	}
	err := walkFn(node, lvl, ce)
	if err != nil {
		return err
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		err = newWalk(child, walkFn, lvl+1, err)
		if errors.Is(err, ErrSkipThread) {
			err = nil
			continue
		} else if err != nil {
			return err
		}
	}
	return nil
}

var ErrSkipThread = errors.New("skip this Thread")

type NewThreadWalkFn func(t *Thread, level int, currentErr error) error

// Implement interface to be able to sort threads by newest (max UID)
type ByUID []*Thread

func getMaxUID(thread *Thread) uint32 {
	// TODO: should we make this part of the Thread type to avoid recomputation?
	var Uid uint32

	_ = thread.Walk(func(t *Thread, _ int, currentErr error) error {
		if t.Uid > Uid {
			Uid = t.Uid
		}
		return nil
	})
	return Uid
}

func (s ByUID) Len() int {
	return len(s)
}

func (s ByUID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByUID) Less(i, j int) bool {
	maxUID_i := getMaxUID(s[i])
	maxUID_j := getMaxUID(s[j])
	return maxUID_i < maxUID_j
}

func SortThreadsBy(toSort []*Thread, sortBy []uint32) {
	// build a map from sortBy
	uidMap := make(map[uint32]int)
	for i, uid := range sortBy {
		uidMap[uid] = i
	}
	// sortslice of toSort with less function of indexing the map sortBy
	sort.Slice(toSort, func(i, j int) bool {
		return uidMap[getMaxUID(toSort[i])] < uidMap[getMaxUID(toSort[j])]
	})
}
