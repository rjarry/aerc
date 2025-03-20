package imap

import (
	"slices"
	"sort"
	"sync"
)

type SeqMap struct {
	lock sync.Mutex
	// map of IMAP sequence numbers to message UIDs
	m []uint32
}

// Initialize sets the initial seqmap of the mailbox
func (s *SeqMap) Initialize(uids []uint32) {
	s.lock.Lock()
	s.m = make([]uint32, len(uids))
	copy(s.m, uids)
	s.sort()
	s.lock.Unlock()
}

func (s *SeqMap) Size() int {
	s.lock.Lock()
	size := len(s.m)
	s.lock.Unlock()
	return size
}

// Get returns the UID of the given seqnum
func (s *SeqMap) Get(seqnum uint32) (uint32, bool) {
	if int(seqnum) > s.Size() || seqnum < 1 {
		return 0, false
	}
	s.lock.Lock()
	uid := s.m[seqnum-1]
	s.lock.Unlock()
	return uid, true
}

// Put adds a UID to the slice. Put should only be used to add new messages
// into the slice
func (s *SeqMap) Put(uid uint32) {
	s.lock.Lock()
	for _, n := range s.m {
		if n == uid {
			// We already have this UID, don't insert it.
			s.lock.Unlock()
			return
		}
	}
	s.m = append(s.m, uid)
	s.sort()
	s.lock.Unlock()
}

func (s *SeqMap) Snapshot(uids []uint32) map[uint32]uint32 {
	snapshot := make(map[uint32]uint32)
	s.lock.Lock()
	for num, uid := range s.m {
		if slices.Contains(uids, uid) {
			// IMAP sequence numbers start at 1
			seqNum := uint32(num) + 1
			snapshot[seqNum] = uid
		}
	}
	s.lock.Unlock()
	return snapshot
}

// Pop removes seqnum from the SeqMap. seqnum must be a valid seqnum, ie
// [1:size of mailbox]
func (s *SeqMap) Pop(seqnum uint32) (uint32, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if int(seqnum) > len(s.m) || seqnum < 1 {
		return 0, false
	}
	uid := s.m[seqnum-1]
	s.m = append(s.m[:seqnum-1], s.m[seqnum:]...)
	return uid, true
}

// sort sorts the slice in ascending UID order. See:
// https://datatracker.ietf.org/doc/html/rfc3501#section-2.3.1.2
func (s *SeqMap) sort() {
	// Always be sure the SeqMap is sorted
	sort.Slice(s.m, func(i, j int) bool {
		return s.m[i] < s.m[j]
	})
}
