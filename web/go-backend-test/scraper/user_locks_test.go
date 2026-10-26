package scraper

import (
	"testing"
)

func TestUserLocksAreIndependentAndReleased(t *testing.T) {
	s := &BUCTScraper{}
	unlockA := s.lockUser("a")
	unlockedB := make(chan struct{})
	go func() { unlock := s.lockUser("b"); unlock(); close(unlockedB) }()
	<-unlockedB
	unlockA()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.userLocks) != 0 {
		t.Fatal("per-user locks leaked after use")
	}
}
