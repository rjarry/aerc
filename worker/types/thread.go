package types

import (
	"errors"
	"fmt"
)

type Thread struct {
	Uid         uint32
	Parent      *Thread
	PrevSibling *Thread
	NextSibling *Thread
	FirstChild  *Thread

	Hidden  bool // if this flag is set the message isn't rendered in the UI
	Deleted bool // if this flag is set the message was deleted
}

func (t *Thread) AddChild(child *Thread) {
	t.insertCmp(child, func(child, iter *Thread) bool { return true })
}

func (t *Thread) OrderedInsert(child *Thread) {
	t.insertCmp(child, func(child, iter *Thread) bool { return child.Uid > iter.Uid })
}

func (t *Thread) insertCmp(child *Thread, cmp func(*Thread, *Thread) bool) {
	if t.FirstChild == nil {
		t.FirstChild = child
	} else {
		start := &Thread{Uid: t.FirstChild.Uid, NextSibling: t.FirstChild}
		var iter *Thread
		for iter = start; iter.NextSibling != nil && cmp(child, iter); iter = iter.NextSibling {
		}
		child.NextSibling = iter.NextSibling
		iter.NextSibling = child
		t.FirstChild = start.NextSibling
	}
	child.Parent = t
}

func (t *Thread) Walk(walkFn NewThreadWalkFn) error {
	err := newWalk(t, walkFn, 0, nil)
	if err == ErrSkipThread {
		return nil
	}
	return err
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
		if err == ErrSkipThread {
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

//Implement interface to be able to sort threads by newest (max UID)
type ByUID []*Thread

func getMaxUID(thread *Thread) uint32 {
	// TODO: should we make this part of the Thread type to avoid recomputation?
	var Uid uint32

	thread.Walk(func(t *Thread, _ int, currentErr error) error {
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
