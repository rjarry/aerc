// Author: Jim Idle - jimi@idle.ws / jimi@gatherstars.com
// SPDX-License-Identifier: Apache-2.0

package jwz

import "sort"

// ThreadLess specifies the signature of a function that compares two Threadables in some way you define,
// such as comparing dates in the emails they contain. Note your function should be able to handle Dummy
// Threadables in some sensible way, such as using the child of the Dummy for the sort parameters.
//
// ThreadLess reports whether the Threadable t1 must sort before the Threadable t2.
//
// If both ThreadLess(t1, t2) and ThreadLess(t2, t1) are false, then t1 and t2 are considered equal.
// Sort may place equal elements in any order in the final result.
//
// ThreadLess must describe a transitive ordering:
//
//   - if both ThreadLess(t1, t2) and ThreadLess(t2, t3) are true, then ThreadLess(t1, t3) must be true as well.
//   - if both ThreadLess(t1, t2) and ThreadLess(t2, t3) are false, then ThreadLess(t1, t3) must be false as well.
type ThreadLess func(t1 Threadable, t2 Threadable) bool

// WalkFunction specifies the signature of a function that can be called by the generic Walk utility function.
// As well as being passed the current Threadable in the walk, the function  will be passed an interface of your
// choosing, which is passed in to the walk function and then propagated  to each call of this function. T
//
// The Walk will not interact with the any, just keep it accessible to your walk function, hence you
// can pass nil if you do not need it.
//
// If your searcher function returns true, then the tree walk will end where it is
type WalkFunction func(t Threadable, u any) (bool, error)

// Count will traverse the supplied Threadable and store the count of Threadables contained within it in the
// given counter location.
//
// Note that any Dummy placeholder nodes are excluded from the count.
func Count(root Threadable, counter *int) {
	if root == nil {
		return
	}

	for node := root; node != nil; node = node.GetNext() {

		if c := node.GetChild(); c != nil {
			// Count children of the current one first then
			//
			Count(c, counter)
		}

		// Only count this one if it is not a dummy placeholder
		//
		if !node.IsDummy() {
			*counter++
		}
	}
}

// Sort will create order from the chaos created by threading a set of emails, that even if given as input
// in a specific order, will be threaded in whatever order the go data structures happen to spit out - which
// will usually be a different order each time.
//
// Note that Sort will not change the embedded nature of the Threads. As in, while the child of a particular
// Threadable can, and usually will, be changed, the new child will belong to the set of next Threadables that
// its original child pointer belonged to. If sorting by date for instance, the set of reply threads/emails
// will be ordered such that the replies are in date order, which is what you want. However, the set of replies
// will remain the same, as the song goes.
//
// Note that you should use the returned Threadable as the new root of your threads, as your old one is
// very likely to have been moved halfway down the top level chain that you passed in. That will confuse the
// BeJesus out of you. I know it did me for a minute.
//
// So, all the chains will be sorted by asking your supplied ThreadLess function whether the first Threadable
// it gives you should sort before the second one it gives you. This makes the sort trivial for you to sort
// the Threadable set and avoids you having to get your head around following the chain of pointers etc.
func Sort(threads Threadable, by ThreadLess) Threadable {
	// Guard against stupidity
	//
	if threads == nil {
		return nil
	}

	// If this node has no next pointers, then it is the only element in this current chain, so it is
	// sorted by default
	//
	if threads.GetNext() == nil {
		return threads
	}

	// Now we sort the chain of current pointers. The easiest way is to convert to a slice then
	// sort the slice. This is because after sorting, the child of the node above should become the
	// first element of the sorted slice
	//
	s := make([]Threadable, 0, 42)
	for current := threads; current != nil; current = current.GetNext() {

		// If the current node we are inspecting has a child, then sort that first
		//
		if c := current.GetChild(); c != nil {
			current.SetChild(Sort(c, by))
		}
		s = append(s, current)
	}

	// We can now sort the slice of next pointers at this level. Note that the child of this node
	// will already have been sorted, so if it is used to get the date or something like that
	// - because this node is a Dummy - then it will be OK to use it as it will be in the correct
	// order already
	//
	sort.Slice(s, func(i, j int) bool {
		return by(s[i], s[j])
	})

	// And we now rebuild the chain from the slice
	//
	l := len(s) - 1
	for i := range l {
		s[i].SetNext(s[i+1])
	}

	// Last element in the slice no longer has a current of course
	//
	s[l].SetNext(nil)

	// And the new child of the node above is the first of the newly sorted (or at the top level the new root)
	//
	newChild := s[0]
	s = nil

	// And return the new child for the node above us
	//
	return newChild
}

// Walk allows the caller to execute some function against each node in the tree, while avoiding dealing with the
// internal structure of the Threadable tree. It is easy to get tree walk code wrong, and while this generic walk
// will probably not do everything that everyone wants - such as cause Leeds United to beat Manchester United - it
// is likely to work for most cases.
//
// Walk will call your WalkFunction for each node in the tree and will supply the node, and an interface value
// (which can be nil) that you can supply for your own tracking if simple walk is not enough for you.
//
// The walker will perform depth first search if asked, by passing parameter isDepth=true
func Walk(isDepth bool, tree Threadable, f WalkFunction, u any) error {
	// Guard against stupidity
	//
	if tree == nil {
		return nil
	}

	// A depth first search means we descend in to the children until there aren't any, then traverse the
	// next pointers. A breadth first search means we traverse all the next pointers then descend in to the children
	//
	if isDepth {
		for current := tree; current != nil; current = current.GetNext() {

			// If the current node we are inspecting has a child, then call with that first
			//
			if c := current.GetChild(); c != nil {
				if err := Walk(isDepth, c, f, u); err != nil {
					return err
				}
			}

			// Now call the user function on this current node, which of course means we eventually call them all
			//
			stop, err := f(current, u)
			if err != nil || stop {
				return err
			}
		}
	} else {

		// This is breadth first, which means we call the function with all the Threadables in the next
		// chain, then we call their children if they have any
		//
		for current := tree; current != nil; current = current.GetNext() {

			// Call the function on each next in turn
			//
			stop, err := f(current, u)
			if err != nil || stop {
				return err
			}
		}
		for current := tree; current != nil; current = current.GetNext() {
			// If the current node we are inspecting has a child, then walk it breadth wise
			//
			if c := current.GetChild(); c != nil {
				if err := Walk(isDepth, c, f, u); err != nil {
					return err
				}
			}
		}
	}
	// We are finished with this bit
	//
	return nil
}
