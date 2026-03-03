package store

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// ErrAlreadyExists is returned when an item already exists in the store.
var ErrAlreadyExists = errors.New("item already exists")

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

// MockCustomerStore is an in-memory implementation of CustomerStore for testing.
type MockCustomerStore struct {
	mu        sync.RWMutex
	customers map[string]*models.Customer
}

func NewMockCustomerStore() *MockCustomerStore {
	return &MockCustomerStore{customers: make(map[string]*models.Customer)}
}

func (m *MockCustomerStore) CreateCustomer(customer *models.Customer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.customers[customer.ID]; exists {
		return ErrAlreadyExists
	}
	c := *customer
	m.customers[customer.ID] = &c
	return nil
}

func (m *MockCustomerStore) GetCustomer(id string) (*models.Customer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if c, ok := m.customers[id]; ok {
		copy := *c
		return &copy, nil
	}
	return nil, nil
}

func (m *MockCustomerStore) ListCustomers() ([]models.Customer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]models.Customer, 0, len(m.customers))
	for _, c := range m.customers {
		result = append(result, *c)
	}
	return result, nil
}

// MockSettingsStore is an in-memory implementation of SettingsStore for testing.
type MockSettingsStore struct {
	mu       sync.RWMutex
	settings map[string]*models.Setting
}

func NewMockSettingsStore() *MockSettingsStore {
	return &MockSettingsStore{settings: make(map[string]*models.Setting)}
}

func (m *MockSettingsStore) GetSetting(key string) (*models.Setting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.settings[key]; ok {
		copy := *s
		return &copy, nil
	}
	return nil, nil
}

func (m *MockSettingsStore) PutSetting(setting *models.Setting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := *setting
	m.settings[setting.Key] = &s
	return nil
}
