package store

import (
	"bili-auth-backend/internal/model"
	"log/slog"
	"sync"
	"time"
)

type MemorySessionStore struct {
	mu              sync.RWMutex
	sessions        map[string]*model.LoginSession
	cleanupInterval time.Duration
	stopCh          chan struct{}
	logger          *slog.Logger
}

func NewMemorySessionStore(cleanupInterval time.Duration, logger *slog.Logger) *MemorySessionStore {
	return &MemorySessionStore{
		sessions:        make(map[string]*model.LoginSession),
		cleanupInterval: cleanupInterval,
		stopCh:          make(chan struct{}),
		logger:          logger,
	}
}

func (m *MemorySessionStore) Save(session *model.LoginSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := *session
	m.sessions[session.SessionID] = &clone
	return nil
}

func (m *MemorySessionStore) Get(sessionID string) (*model.LoginSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	clone := *s
	return &clone, true
}

func (m *MemorySessionStore) Delete(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
	return nil
}

func (m *MemorySessionStore) List() ([]*model.LoginSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*model.LoginSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		clone := *s
		result = append(result, &clone)
	}
	return result, nil
}

func (m *MemorySessionStore) StartCleanup() {
	ticker := time.NewTicker(m.cleanupInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				m.cleanupExpired()
			case <-m.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

func (m *MemorySessionStore) StopCleanup() {
	close(m.stopCh)
}

func (m *MemorySessionStore) cleanupExpired() {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for sid, s := range m.sessions {
		if now.After(s.ExpiresAt) {
			delete(m.sessions, sid)
			m.logger.Info("session expired and cleaned", "session_id", sid)
		}
	}
}
