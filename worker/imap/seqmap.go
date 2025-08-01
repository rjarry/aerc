package imap

import (
	"slices"
	"sync"

	"git.sr.ht/~rjarry/aerc/lib/log"
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
	if slices.Contains(s.m, uid) {
		// We already have this UID, don't insert it.
		s.lock.Unlock()
		return
	}
	s.m = append(s.m, uid)
	s.sort()
	s.lock.Unlock()
}

// Take a snapshot of the SequenceNumber=>UID mappings for the given UIDs,
// remove those UIDs from the SeqMap, and return the snapshot it to the caller,
// as well as the loweest sequence number it contains.
func (s *SeqMap) Snapshot(uids []uint32) (map[uint32]uint32, uint32) {
	// Take the snapshot.
	snapshot := make(map[uint32]uint32)
	var minSequenceNum uint32 = 0
	var snapshotSeqNums []uint32
	s.lock.Lock()
	for num, uid := range s.m {
		if slices.Contains(uids, uid) {
			// IMAP sequence numbers start at 1
			seqNum := uint32(num) + 1
			snapshotSeqNums = append(snapshotSeqNums, seqNum)
			if minSequenceNum == 0 {
				minSequenceNum = seqNum
			}
			snapshot[seqNum] = uid
		}
	}
	s.lock.Unlock()

	// Remove the snapshotted mappings from the sequence; we need to do it from
	// the highest to the lowest key since a SeqMap.Pop moves all the items on
	// the right of the popped sequence number by one position to the left.
	for i := len(snapshotSeqNums) - 1; i >= 0; i-- {
		_, ok := s.Pop(snapshotSeqNums[i])
		if !ok {
			log.Errorf("Unable to pop %d from SeqMap", snapshotSeqNums[i])
		}
	}

	return snapshot, minSequenceNum
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
	slices.Sort(s.m)
}
