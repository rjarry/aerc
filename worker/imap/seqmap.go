package imap

import "sync"

type SeqMap struct {
	lock sync.Mutex
	// map of IMAP sequence numbers to message UIDs
	m map[uint32]uint32
}

func (s *SeqMap) Size() int {
	s.lock.Lock()
	size := len(s.m)
	s.lock.Unlock()
	return size
}

func (s *SeqMap) Get(seqnum uint32) (uint32, bool) {
	s.lock.Lock()
	uid, found := s.m[seqnum]
	s.lock.Unlock()
	return uid, found
}

func (s *SeqMap) Put(seqnum, uid uint32) {
	s.lock.Lock()
	if s.m == nil {
		s.m = make(map[uint32]uint32)
	}
	s.m[seqnum] = uid
	s.lock.Unlock()
}

func (s *SeqMap) Pop(seqnum uint32) (uint32, bool) {
	s.lock.Lock()
	uid, found := s.m[seqnum]
	if found {
		delete(s.m, seqnum)
	}
	s.lock.Unlock()
	return uid, found
}

func (s *SeqMap) Clear() {
	s.lock.Lock()
	s.m = make(map[uint32]uint32)
	s.lock.Unlock()
}
