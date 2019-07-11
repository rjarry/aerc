// Package uidstore provides a concurrency-safe two-way mapping between UIDs
// used by the UI and arbitrary string keys as used by different mail backends.
//
// Multiple Store instances can safely be created and the UIDs that they
// generate will be globally unique.
package uidstore

import (
	"sync"
	"sync/atomic"
)

var nextUID uint32 = 1

// Store holds a mapping between application keys and globally-unique UIDs.
type Store struct {
	keyByUID map[uint32]string
	uidByKey map[string]uint32
	m        sync.Mutex
}

// NewStore creates a new, empty Store.
func NewStore() *Store {
	return &Store{
		keyByUID: make(map[uint32]string),
		uidByKey: make(map[string]uint32),
	}
}

// GetOrInsert returns the UID for the provided key. If the key was already
// present in the store, the same UID value is returned. Otherwise, the key is
// inserted and the newly generated UID is returned.
func (s *Store) GetOrInsert(key string) uint32 {
	s.m.Lock()
	defer s.m.Unlock()
	if uid, ok := s.uidByKey[key]; ok {
		return uid
	}
	uid := atomic.AddUint32(&nextUID, 1)
	s.keyByUID[uid] = key
	s.uidByKey[key] = uid
	return uid
}

// GetKey returns the key for the provided UID, if available.
func (s *Store) GetKey(uid uint32) (string, bool) {
	s.m.Lock()
	defer s.m.Unlock()
	key, ok := s.keyByUID[uid]
	return key, ok
}

// RemoveUID removes the specified UID from the store.
func (s *Store) RemoveUID(uid uint32) {
	s.m.Lock()
	defer s.m.Unlock()
	key, ok := s.keyByUID[uid]
	if ok {
		delete(s.uidByKey, key)
	}
	delete(s.keyByUID, uid)
}
