// Author: Jim Idle - jimi@idle.ws / jimi@gatherstars.com
// SPDX-License-Identifier: Apache-2.0

package jwz

// NoThreadableError indicates that the container was not in a valid state when a utility function was called
// against it
type NoThreadableError struct{}

func (e NoThreadableError) Error() string {
	return "Container is not a root (has parent), but contains no Threadable element"
}

// threadContainer  is used to encapsulate a Threadable implementation. It holds some intermediate state used while
// threading.  This is private to the module and is used while performing the threading operation, then discarded.
type threadContainer struct {
	// threadable contains the base of the threadable items in this instance of the struct
	//
	threadable Threadable

	// parent shows which threadable item is the parent of this container, which can be nil for a root node
	//
	parent *threadContainer

	// child shows which threadable is the child of this container
	//
	child *threadContainer

	// next shows which threadable is the sibling of this container
	//
	next *threadContainer

	// forID holds the message id that this container is holding. This is only useful for
	// when we don't ever see the actual email (it is referenced by one email, but we don't
	// have the email as we are parsing a partial set)
	//
	forID string
}

// flush copies the ThreadContainer tree structure down into the underlying
// Threadable elements. I.E. make the IThreadable tree look like
// the ThreadContainer tree.
func (tc *threadContainer) flush() error {
	// Only a root node can have no threadable element - if we have a parent, then
	// we are not a root node.
	//
	if tc.parent != nil && tc.threadable == nil {
		return NoThreadableError{}
	}

	tc.parent = nil

	if tc.threadable != nil {
		if tc.child == nil {
			tc.threadable.SetChild(nil)
		} else {
			tc.threadable.SetChild(tc.child.threadable)
		}
	}

	if tc.child != nil {
		_ = tc.child.flush()
		tc.child = nil
	}

	if tc.threadable != nil {
		if tc.next == nil {
			tc.threadable.SetNext(nil)
		} else {
			tc.threadable.SetNext(tc.next.threadable)
		}
	}

	if tc.next != nil {
		_ = tc.next.flush()
		tc.next = nil
	}

	tc.threadable = nil

	return nil
}

// findChild returns true if child is under self's tree. This is used for
// detecting circularities in the references header.
func (tc *threadContainer) findChild(target *threadContainer) bool {
	found := false
	for t := tc.child; t != nil; t = t.next {
		found = found || t == target || t.findChild(target)
	}
	return found
}

// reverseChildren does what it implies and reverses the order of the child elements
func (tc *threadContainer) reverseChildren() {
	if tc.child != nil {

		var kid, prev, rest *threadContainer

		// Reverse the children of this one
		//
		prev = nil
		kid = tc.child
		rest = kid.next
		for kid != nil {

			kid.next = prev

			prev = kid
			kid = rest
			if rest != nil {
				rest = rest.next
			}
		}
		tc.child = prev

		// Now reverse the children of this one's children
		//
		for kid = tc.child; kid != nil; kid = kid.next {
			kid.reverseChildren()
		}
	}
}

func (tc *threadContainer) fillDummy(t Threadable) {
	// Only a root node can have no threadable element - if we have a message id, then
	// we are not a root node.
	//
	if tc.threadable == nil && tc.forID != "" {
		tc.threadable = t.MakeDummy(tc.forID)
	}

	// Depth first, but it does not matter
	//
	if tc.child != nil {
		tc.child.fillDummy(t)
	}

	// Simple walk this one, no need to save stack space by following the next chain here
	//
	if tc.next != nil {
		tc.next.fillDummy(t)
	}
}
