package types

import (
	"errors"
	"fmt"
	"sort"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
)

type Thread struct {
	Uid         models.UID
	Parent      *Thread
	PrevSibling *Thread
	NextSibling *Thread
	FirstChild  *Thread

	Hidden  int  // if this flag is not zero the message isn't rendered in the UI
	Deleted bool // if this flag is set the message was deleted

	// if this flag is set the message is the root of an incomplete thread
	Dummy bool

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
func (t *Thread) Uids() []models.UID {
	if t == nil {
		return nil
	}
	uids := make([]models.UID, 0)
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
	var parent models.UID
	if t.Parent != nil {
		parent = t.Parent.Uid
	}
	var next models.UID
	if t.NextSibling != nil {
		next = t.NextSibling.Uid
	}
	var child models.UID
	if t.FirstChild != nil {
		child = t.FirstChild.Uid
	}
	return fmt.Sprintf(
		"[%s] (parent:%s, next:%s, child:%s)",
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

func getMaxUID(thread *Thread) models.UID {
	// TODO: should we make this part of the Thread type to avoid recomputation?
	var Uid models.UID

	_ = thread.Walk(func(t *Thread, _ int, currentErr error) error {
		if t.Deleted || t.Hidden > 0 {
			return nil
		}
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

func getMaxValue(thread *Thread, uidMap map[models.UID]int) int {
	var max int

	_ = thread.Walk(func(t *Thread, _ int, currentErr error) error {
		if t.Deleted || t.Hidden > 0 {
			return nil
		}
		if uidMap[t.Uid] > max {
			max = uidMap[t.Uid]
		}
		return nil
	})
	return max
}

func SortThreadsBy(toSort []*Thread, sortBy []models.UID) {
	// build a map from sortBy
	uidMap := make(map[models.UID]int)
	for i, uid := range sortBy {
		uidMap[uid] = i
	}
	// sortslice of toSort with less function of indexing the map sortBy
	sort.Slice(toSort, func(i, j int) bool {
		return getMaxValue(toSort[i], uidMap) < getMaxValue(toSort[j], uidMap)
	})
}
