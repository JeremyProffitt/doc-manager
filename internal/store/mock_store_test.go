package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// MockUserStore is an in-memory implementation of UserStore for testing.
type MockUserStore struct {
	mu    sync.RWMutex
	users map[string]*models.User
}

func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users: make(map[string]*models.User),
	}
}

func (m *MockUserStore) GetUser(email string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	// Return a copy to prevent mutation
	copy := *u
	return &copy, nil
}

func (m *MockUserStore) CreateUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.users[user.Email]; exists {
		return fmt.Errorf("user already exists: %s", user.Email)
	}
	copy := *user
	m.users[user.Email] = &copy
	return nil
}

// MockSessionStore is an in-memory implementation of SessionStore for testing.
type MockSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*models.Session
}

func NewMockSessionStore() *MockSessionStore {
	return &MockSessionStore{
		sessions: make(map[string]*models.Session),
	}
}

func (m *MockSessionStore) CreateSession(session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *session
	m.sessions[session.Token] = &copy
	return nil
}

func (m *MockSessionStore) GetSession(token string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[token]
	if !ok {
		return nil, nil
	}
	// Check expiration
	if s.ExpiresAt < time.Now().Unix() {
		return nil, nil
	}
	copy := *s
	return &copy, nil
}

func (m *MockSessionStore) DeleteSession(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
	return nil
}
