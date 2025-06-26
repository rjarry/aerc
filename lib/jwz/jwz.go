// Package jwz is an implementation of the email threading algorithm created by Jamie Zawinski and explained by him
// at: https://www.jwz.org/doc/threading.html
//
// This package was created by cribbing from the code at:
//
//	https://www.jwz.org/doc/threading.html#:~:text=grendel-1999-05-14.tar.gz
//
// from the Java source code in view/Threader.java - it contains no ham and cheese sandwiches.
//
// The code, interface etc. was obviously adapted in to Go form, though where possible, the code reflects the
// original Java if it is not too ungolike.
//
// Author: Jim Idle - jimi@idle.ws / jimi@gatherstars.com
// SPDX-License-Identifier: Apache-2.0
//
// See the LICENSE file, sit down, have a scone.

package jwz

import (
	"errors"
	"fmt"
)

// Threader arranges a set of messages into a thread hierarchy, by references.
type Threader struct {
	rootNode     *threadContainer
	idTable      map[string]*threadContainer
	bogusIDCount int
}

// NewThreader returns an instance of the Threader struct, that is ready to attack
// your Threadable
//
//goland:noinspection GoUnusedExportedFunction
func NewThreader() *Threader {
	t := &Threader{
		idTable: make(map[string]*threadContainer),
	}
	return t
}

// Thread will create a threadable organized  so that the root node
// is the original reference, creating dummy placeholders for the emails
// we don't have yet
func (t *Threader) Thread(threadable Threadable) (Threadable, error) {
	if threadable == nil {
		return nil, nil
	}

	// Build a thread container from this single email
	//
	if !threadable.IsDummy() {
		if err := t.buildContainer(threadable); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("cannot thread a single email with a dummy root")
	}

	var err error

	// Organize the root set from what we have
	//
	t.rootNode, err = t.findRootSet()
	if err != nil {
		return nil, err
	}

	// We no longer need the map - probably no real need to blank it here, but the original Java code did that,
	// and it won't harm to let the GC reclaim this in case our caller keeps the *Threader around for some reason
	//
	t.idTable = nil

	// We do this to avoid flipping the input order each time through.
	//
	t.rootNode.reverseChildren()

	// There should not be a next in the root of a conversation thread
	//
	if t.rootNode.next != nil {
		return nil, fmt.Errorf("root node contains a next and should not: %#v", t.rootNode)
	}

	// Because the result of this function is a tree that does actually contain dummies for missing references
	// we need to add a dummy threadable for any node that does not yet have one. Then we can flush the chain
	// of containers in to the threadable
	//
	t.rootNode.fillDummy(threadable)

	var result Threadable
	if t.rootNode.child != nil {
		result = t.rootNode.child.threadable
	}

	// Flush the tree structure of each element of the root set down into
	// their underlying Threadables
	//
	_ = t.rootNode.flush()
	t.rootNode = nil

	return result, nil
}

// ThreadSlice will thread the set of messages contained within threadableSlice.
// The Threadable returned is the new first element of the root set.
func (t *Threader) ThreadSlice(threadableSlice []Threadable) (Threadable, error) {
	if len(threadableSlice) == 0 {
		return nil, nil
	}

	// Iterate all the Threadable represented by the root and build the
	// threadContainer from them
	//
	for _, nt := range threadableSlice {
		if !nt.IsDummy() {
			if err := t.buildContainer(nt); err != nil {
				return nil, err
			}
		}
	}
	return t.threadRoot()
}

// ThreadRoot will thread the set of messages provided by ThreadableRoot.
// The Threadable returned is the new first element of the root set.
func (t *Threader) ThreadRoot(threadableRoot ThreadableRoot) (Threadable, error) {
	if threadableRoot == nil {
		return nil, nil
	}

	// Iterate all the Threadable represented by the root and build the
	// threadContainer from them
	//
	for threadableRoot.Next() {
		nt := threadableRoot.Get()
		if !nt.IsDummy() {
			if err := t.buildContainer(nt); err != nil {
				return nil, err
			}
		}
	}
	return t.threadRoot()
}

func (t *Threader) threadRoot() (Threadable, error) {
	var err error

	// Organize the root set from what we have
	//
	t.rootNode, err = t.findRootSet()
	if err != nil {
		return nil, err
	}

	// We no longer need the map - probably no real need to blank it here, but the original Java code did that,
	// and it won't harm to let the GC reclaim this in case our caller keeps the *Threader around for some reason
	//
	t.idTable = nil

	// Get rid of any empty containers. They should no longer needed
	//
	t.pruneEmptyContainers(t.rootNode)

	// We do this so to avoid flipping the input order each time through.
	//
	t.rootNode.reverseChildren()

	// We might need to sort on subjects, so let's process them
	//
	t.gatherSubjects()

	// There should not be a next in the root of a conversation thread
	//
	if t.rootNode.next != nil {
		return nil, fmt.Errorf("root node contains a next and should not: %#v", t.rootNode)
	}

	for r := t.rootNode.child; r != nil; r = r.next {
		// If this direct child of the root node has no threadable in it,
		// manufacture a dummy container to bind its children together.
		// Note that these dummies can only ever occur as elements of
		// the root set.
		//
		if r.threadable == nil {
			r.threadable = r.child.threadable.MakeDummy(r.forID)
		}
	}

	var result Threadable
	if t.rootNode.child != nil {
		result = t.rootNode.child.threadable
	}

	// Flush the tree structure of each element of the root set down into
	// their underlying Threadables
	//
	_ = t.rootNode.flush()
	t.rootNode = nil

	return result, nil
}

// buildContainer() does three things:
//
//   - It walks the tree of Threadable, and wraps each in a
//     threadContainer object.
//   - It indexes each threadContainer object in the idTable, under
//     the message ID of the contained Threadable.
//   - For each of the references within Threadable, it ensures that there
//     is a threadContainer in the table (an empty one, if necessary.)
func (t *Threader) buildContainer(threadable Threadable) error {
	var present bool

	// See if we already have a container for this threadable
	//
	id := threadable.MessageThreadID()
	tid := id

	c, present := t.idTable[id]
	if present {
		// There is already a ThreadContainer in the table for this ID.
		// Under normal circumstances, there will be no IThreadable in it
		// (since it was a forward reference from a References field.)
		//
		// If there is already a threadable in it, then that means there
		// are two IThreadables with the same ID.  Generate a new ID for
		// this one, sigh...  This ID is only used to cause the two entries
		// in the hash table to not stomp each other.
		//
		if c.threadable != nil {
			id = fmt.Sprintf("<Bogus-id:%d>", t.bogusIDCount)
			t.bogusIDCount++
			c = nil
		} else {
			c.threadable = threadable
		}
	}

	// Create a ThreadContainer for this Threadable, and store it in
	// the map
	//
	if c == nil {
		c = &threadContainer{forID: tid}
		c.threadable = threadable
		c.forID = tid
		t.idTable[id] = c
	}

	// Create ThreadContainers for each of the references which don't
	// have them.  Link each of the referenced messages together in the
	// order implied by the references field, unless they are already
	// linked.
	//
	var parentRef, ref *threadContainer

	// Iterate through the references field of the threadable and see if we
	// already have a reference to them in our map. Create one if not
	//
	refs := threadable.MessageThreadReferences()
	for _, refString := range refs {

		ref, present = t.idTable[refString]
		if !present {

			ref = &threadContainer{forID: refString}

			t.idTable[refString] = ref
		}

		// If we have references A B C D, make D be a child of C, etc.,
		// except if they have parents already.
		//
		if parentRef != nil && // there is a parent
			ref.parent == nil && // don't have a parent already
			parentRef != ref && // not a tight loop
			!ref.findChild(parentRef) && // already linked
			!parentRef.findChild(ref) { // not a wide loop

			// Ok, link it into the parent's child list.
			//
			ref.parent = parentRef
			ref.next = parentRef.child
			parentRef.child = ref
		}
		parentRef = ref
	}

	// At this point `parentRef' is set to the container of the last element
	// in the references field.  Make that be the parent of this container,
	// unless doing so would introduce a circularity.
	//
	if parentRef != nil &&
		(parentRef == c ||
			c.findChild(parentRef)) {
		parentRef = nil
	}

	if c.parent != nil {

		// If it has a parent already, that's there because we saw this message
		// in a references field, and presumed a parent based on the other
		// entries in that field.  Now that we have the actual message, we can
		// be more definitive, so throw away the old parent and use this new one.
		// Find this container in the parent's child-list, and unlink it.
		//
		// Note that this could cause this message to now have no parent, if it
		// has no references field, but some message referred to it as the
		// non-first element of its references.  (Which would have been some
		// kind of lie...)
		//
		var rest, prev *threadContainer
		for prev, rest = nil, c.parent.child; rest != nil; {
			if rest == c {
				break
			}
			prev = rest
			rest = rest.next
		}

		if rest == nil {
			return fmt.Errorf("didn't find %#v in parent %#v", c, c.parent)
		}

		if prev == nil {
			c.parent.child = c.next
		} else {
			prev.next = c.next
		}

		c.next = nil
		c.parent = nil
	}

	// If we have a parent, link c into the parent's child list.
	//
	if parentRef != nil {
		c.parent = parentRef
		c.next = parentRef.child
		parentRef.child = c
	}

	// No error
	//
	return nil
}

// findRootSet finds the root set of the threadContainers, and returns a root node.
//
// NB: A container is in the root set if it has no parents.
func (t *Threader) findRootSet() (*threadContainer, error) {
	root := &threadContainer{}
	for _, c := range t.idTable {
		if c.parent == nil {
			if c.next != nil {
				return nil, fmt.Errorf("container has no parent, but has a next value: %#v", c.next)
			}
			c.next = root.child
			root.child = c
		}
	}
	return root, nil
}

// Walk through the threads and discard any empty container objects.
// After calling this, there will only be any empty container objects
// at depth 0, and those will all have at least two kids.
func (t *Threader) pruneEmptyContainers(parent *threadContainer) {
	var prev *threadContainer
	container := parent.child
	var next *threadContainer

	if container != nil {
		next = container.next
	}

	for container != nil {
		switch {
		case container.threadable == nil && container.child == nil:
			// This is an empty container with no kids.  Nuke it.
			//
			// Normally such containers won't occur, but they can show up when
			// two messages have References lines that disagree.  For example,
			// assuming A and B are messages, and 1, 2, and 3 are references for
			// messages we haven't seen:
			//
			//        A has refs: 1 2 3
			//        B has refs: 1 3
			//
			// There is ambiguity whether 3 is a child of 1 or 2.  So,
			// depending on the processing order, we might end up with either
			//
			//        -- 1
			//           |-- 2
			//               |-- 3
			//                   |-- A
			//                   |-- B
			// or
			//        -- 1
			//           |-- 2            <--- non-root childless container
			//           |-- 3
			//               |-- A
			//               |-- B
			//
			if prev == nil {
				parent.child = container.next
			} else {
				prev.next = container.next
			}

			// Set container to prev so that prev keeps its same value
			// the next time through the loop.
			//
			container = prev

		case container.threadable == nil && // expired, and
			container.child != nil && // has kids, and
			(container.parent != nil || // not at root, or
				container.child.next == nil):

			// Expired message with kids.  Promote the kids to this level.
			// Don't do this if we would be promoting them to the root level,
			// unless there is only one kid.
			//
			var tail *threadContainer
			kids := container.child

			// Remove this container from the list, replacing it with `kids'
			//
			if prev == nil {
				parent.child = kids
			} else {
				prev.next = kids
			}

			// Make each child's parent be this level's parent.
			// Make the last child's next be this container's next
			//  - splicing `kids' into the list in place of `container'
			//
			for tail = kids; tail.next != nil; tail = tail.next {
				tail.parent = container.parent
			}
			tail.parent = container.parent
			tail.next = container.next

			// Since we've inserted items in the chain, `next' currently points
			// to the item after them (tail.next); reset that so that we process
			// the newly promoted items the very next time around.
			//
			next = kids

			// Set container to prev so that prev keeps its same value
			// the next time through the loop.
			//
			container = prev

		case container.child != nil:
			// A real message with kids.
			// Iterate over its children, and try to strip out the junk.
			//
			t.pruneEmptyContainers(container)
		}

		// Set up for the next iteration
		//
		prev = container
		container = next
		if container == nil {
			next = nil
		} else {
			next = container.next
		}
	}
}

// If any two members of the root set have the same subject, merge them.
// This is so that messages which don't have References headers at all
// still get threaded (to the extent possible, at least.)
func (t *Threader) gatherSubjects() {
	var count int

	subjTable := make(map[string]*threadContainer)

	for c := t.rootNode.child; c != nil; c = c.next {

		threadable := c.threadable

		// If there is no threadable, this is a dummy node in the root set.
		// Only root set members may be dummies, and they always have at least
		// two kids.  Take the first kid as representative of the subject.
		//
		if threadable == nil {
			threadable = c.child.threadable
		}

		subj := threadable.SimplifiedSubject()
		if subj == "" {
			continue
		}

		old := subjTable[subj]

		// Add this container to the table if:
		//  - There is no container in the table with this subject, or
		//  - This one is a dummy container and the old one is not: the dummy
		//    one is more interesting as a root, so put it in the table instead.
		//  - The container in the table has a "Re:" version of this subject,
		//    and this container has a non-"Re:" version of this subject.
		//    The non-re version is the more interesting of the two.
		//
		if old == nil ||
			(c.threadable == nil && old.threadable != nil) ||
			(old.threadable != nil && old.threadable.SubjectIsReply() &&
				c.threadable != nil && !c.threadable.SubjectIsReply()) {
			subjTable[subj] = c
			count++
		}
	}

	// We are done if the table is empty
	//
	if count == 0 {
		return
	}

	// The subj_table is now populated with one entry for each subject which
	// occurs in the root set.  Now iterate over the root set, and gather
	// together the difference.
	//
	var prev, c, rest *threadContainer

	prev = nil
	c = t.rootNode.child
	rest = c.next

	for c != nil {

		threadable := c.threadable

		// might be a dummy -- see above
		//
		if threadable == nil {
			threadable = c.child.threadable
		}

		subj := threadable.SimplifiedSubject()

		// Don't thread together all subject-less messages; let them dangle.
		//
		if subj != "" {

			old := subjTable[subj]

			if old != c { // Avoid processing ourselves

				// Ok, so now we have found another container in the root set with
				// the same subject.  There are a few possibilities:
				//
				// - If both are dummies, append one's children to the other, and remove
				//   the now-empty container.
				//
				// - If one container is a dummy and the other is not, make the non-dummy
				//   one be a child of the dummy, and a sibling of the other "real"
				//   messages with the same subject (the dummy's children.)
				//
				// - If that container is a non-dummy, and that message's subject does
				//   not begin with "Re:", but *this* message's subject does, then
				//   make this be a child of the other.
				//
				// - If that container is a non-dummy, and that message's subject begins
				//   with "Re:", but *this* message's subject does *not*, then make that
				//   be a child of this one -- they were mis-ordered.  (This happens
				//   somewhat implicitly, since if there are two messages, one with Re:
				//   and one without, the one without will be in the hash table,
				//   regardless of the order in which they were seen.)
				//
				// - Otherwise, make a new dummy container and make both messages be a
				//   child of it.  This catches the both-are-replies and neither-are-
				//   replies cases, and makes them be siblings instead of asserting a
				//   hierarchical relationship which might not be true.
				//
				//   (People who reply to a message without using "Re:" and without using
				//   a References line will break this slightly.  Those people suck.)
				//
				// (It has occurred to me that taking the date or message number into
				// account would be one way of resolving some ambiguous cases,
				// but that's not altogether straightforward either.)
				// JI: You cannot rely on the clock settings being correct on a server/client that sent a message
				//

				// Remove the "second" message from the root set.
				if prev == nil {
					t.rootNode.child = c.next
				} else {
					prev.next = c.next
				}
				c.next = nil

				switch {
				case old.threadable == nil && c.threadable == nil:
					// They're both dummies; merge them.
					//
					var tail *threadContainer
					for tail = old.child; tail != nil && tail.next != nil; tail = tail.next {
					}

					tail.next = c.child
					for tail = c.child; tail != nil; tail = tail.next {
						tail.parent = old
					}
					c.child = nil

				case old.threadable == nil || // old is empty, or
					(c.threadable != nil &&
						c.threadable.SubjectIsReply() && //   c has Re, and
						!old.threadable.SubjectIsReply()): //   old does not.

					// Make this message be a child of the other.
					c.parent = old
					c.next = old.child
					old.child = c

				default:
					// Make the old and new messages be children of a new dummy container.
					// We do this by creating a new container object for old->msg and
					// transforming the old container into a dummy (by merely emptying it),
					// so that the  table still points to the one that is at depth 0
					// instead of depth 1.
					//
					newC := &threadContainer{}

					newC.threadable = old.threadable
					newC.child = old.child
					for tail := newC.child; tail != nil; tail = tail.next {
						tail.parent = newC
					}

					old.threadable = nil
					old.child = nil

					c.parent = old
					newC.parent = old

					// old is now a dummy; make it have exactly two kids, c and newC.
					//
					old.child = c
					c.next = newC
				}

				// we've done a merge, so keep the same `prev' next time around.
				//
				c = prev
			}
		}
		prev = c
		c = rest
		if rest != nil {
			rest = rest.next
		}
	}
}
