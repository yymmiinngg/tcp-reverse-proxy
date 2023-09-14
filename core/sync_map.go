package core

import (
	"sync"
)

// SyncMap SyncMap
type SyncMap struct {
	storage map[interface{}]interface{}
	locker  sync.RWMutex
}

// NewSyncMap NewSyncMap
func MakeSyncMap(size int) *SyncMap {
	m := &SyncMap{
		storage: make(map[interface{}]interface{}, size),
	}
	return m
}

// Load Load
func (s *SyncMap) Get(key interface{}) (interface{}, bool) {
	defer s.locker.RUnlock()
	s.locker.RLock()
	v, ok := s.storage[key]
	return v, ok
}

// Store Store
func (s *SyncMap) Put(key interface{}, value interface{}) {
	defer s.locker.Unlock()
	s.locker.Lock()
	s.storage[key] = value
}

// Delete Delete
func (s *SyncMap) Delete(key interface{}) {
	defer s.locker.Unlock()
	s.locker.Lock()
	delete(s.storage, key)
}

// Range Range
func (s *SyncMap) Range(handler func(key interface{}, value interface{}) bool) {
	defer s.locker.RUnlock()
	s.locker.RLock()
	for k, v := range s.storage {
		if !handler(k, v) {
			break
		}
	}
}

// Size Size
func (s *SyncMap) Size() int {
	defer s.locker.RUnlock()
	s.locker.RLock()
	return len(s.storage)
}
