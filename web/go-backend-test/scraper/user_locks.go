package scraper

import "sync"

type userLock struct {
	mu   sync.Mutex
	refs int
}

func (s *BUCTScraper) lockUser(userID string) func() {
	s.mu.Lock()
	if s.userLocks == nil {
		s.userLocks = make(map[string]*userLock)
	}
	lock := s.userLocks[userID]
	if lock == nil {
		lock = &userLock{}
		s.userLocks[userID] = lock
	}
	lock.refs++
	s.mu.Unlock()
	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		s.mu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(s.userLocks, userID)
		}
		s.mu.Unlock()
	}
}
