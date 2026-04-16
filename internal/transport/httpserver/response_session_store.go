package httpserver

import (
	"sync"
	"time"

	"opencrab/internal/domain"
)

type ResponseSession struct {
	ResponseID string
	SessionID  string
	Model      string
	Messages   []domain.GatewayMessage
	UpdatedAt  time.Time
}

type ResponseSessionStore interface {
	Get(responseID string) (ResponseSession, bool)
	Put(session ResponseSession)
}

type MemoryResponseSessionStore struct {
	mu       sync.RWMutex
	items    map[string]ResponseSession
	maxItems int
}

func NewMemoryResponseSessionStore(maxItems int) *MemoryResponseSessionStore {
	if maxItems <= 0 {
		maxItems = 512
	}
	return &MemoryResponseSessionStore{items: map[string]ResponseSession{}, maxItems: maxItems}
}

func (s *MemoryResponseSessionStore) Get(responseID string) (ResponseSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[responseID]
	return item, ok
}

func (s *MemoryResponseSessionStore) Put(session ResponseSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) >= s.maxItems {
		var oldestKey string
		var oldest time.Time
		for key, item := range s.items {
			if oldestKey == "" || item.UpdatedAt.Before(oldest) {
				oldestKey = key
				oldest = item.UpdatedAt
			}
		}
		if oldestKey != "" {
			delete(s.items, oldestKey)
		}
	}
	s.items[session.ResponseID] = session
}
