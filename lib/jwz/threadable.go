// Author: Jim Idle - jimi@idle.ws / jimi@gatherstars.com
// SPDX-License-Identifier: Apache-2.0

package jwz

import "time"

// ThreadableRoot is an interface that supports traversing a set of Threadables in some arbitrary
// way - for instance if they are in some kind of tree structure, the traversal can be
// hidden behind the interface
//
// JI - Although it might be useful to support incoming tree structures, all the
// Next and Get are doing is keeping a pointer if the input is a []Threadable. So we also
// have a function that just accepts that as input as well as one that accepts a ThreadableRoot
type ThreadableRoot interface {
	// Next causes an internal iterator over your internal representation of Threadable
	// elements to either be created and pointing to the next element, or to simply
	// advance to the next element if there is one. It returns true if another element
	// is available and false if there are no more beans.
	//
	Next() bool

	// Get returns the next available Threadable from your internal storage.
	// Note that this func should not be called without a prior call to Next and your
	// implementation can assume that.
	//
	Get() Threadable
}

// Threadable is an interface which can be implemented by any go type, which will then
// allow it to be threaded.
type Threadable interface {
	// MessageThreadID returns a string identifying this message.
	// Generally this will be a representation of the contents of the
	// Message-ID header.
	//
	MessageThreadID() string

	// MessageThreadReferences returns the IDs of the set of messages referenced by this one.
	// This list should be ordered from oldest-ancestor to youngest-ancestor. However, the returned
	// tree can be sorted however you like.
	//
	MessageThreadReferences() []string

	// Subject returns the subject line of the threadable with no manipulation of Re: Re: etc.
	//
	Subject() string

	// SimplifiedSubject - provides a threadable subject string.
	//
	// When no references are present, subjects will be used to thread together
	// messages.  This method should return a threadable subject: two messages
	// with the same simplifiedSubject will be considered to belong to the same
	// thread.  This string should not have `Re:' on the front, and may have
	// been simplified in whatever other ways seem appropriate.
	//
	// This is a String of Unicode characters, and should have had any encodings -
	// such as RFC 2047 charset encodings - removed first.
	//
	// If you aren't interested in threading by subject at all, return "".
	//
	SimplifiedSubject() string

	// SubjectIsReply indicates whether the original subject was one that appeared to be a reply
	// I.E. it had a `Re:' or some other indicator that lets you determine that.  When threading by subject,
	// this property is used to tell whether two messages appear to be siblings,
	// or in a parent/child relationship.
	//
	SubjectIsReply() bool

	// SetNext is called after the proper thread order has been computed,
	// and will be called on each Threadable in the chain, to set up the proper tree
	// structure.
	//
	SetNext(next Threadable)

	// SetChild is called after the proper thread order has been computed,
	// and will be called on each Threadable in the chain, to set up the proper tree
	// structure.
	//
	SetChild(kid Threadable)

	// SetParent is not called by the jwz algorithm and if you do not need the pointer in your
	// implementation, then you can implement it as a null function. It can be useful when using
	// the Walk utility method though
	//
	SetParent(parent Threadable)

	// GetNext just makes it easier to navigate through the threads after they are built,
	// but you don't have to use this if you have a better way
	//
	GetNext() Threadable

	// GetChild just makes it easier to navigate through the threads after they are built,
	// but you don't have to use this if you have a better way
	//
	GetChild() Threadable

	// GetParent just makes it easier to navigate through the threads after they are built,
	// but you don't have to use this if you have no need for it
	//
	GetParent() Threadable

	// GetDate is not used by the threading algorithm, but implementing this function may make
	// your own tree walking routines and sorting methods easier to implement.
	// It should return the Date associated with the Threadable
	//
	GetDate() time.Time

	// MakeDummy creates a dummy parent object.
	//
	// With some set of messages, the only way to achieve proper threading is
	// to introduce an element into the tree which represents messages which are
	// not present in the set: for example, when two messages share a common
	// ancestor, but that ancestor is not in the set. This method is used to
	// make a placeholder for those sorts of ancestors. It should return
	// a Threadable type.  The SetNext() and SetChild() funcs
	// will be used on this placeholder, as either the object or the argument,
	// just as for other elements of the tree.
	//
	MakeDummy(forID string) Threadable

	// IsDummy should return true of dummy messages, false otherwise.
	// It is legal to pass dummy messages within your input;
	// the isDummy() method is the mechanism by which they are noted and ignored.
	//
	IsDummy() bool
}
