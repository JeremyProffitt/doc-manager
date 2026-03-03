package store

import (
	"errors"
	"fmt"
	"sort"
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

func (m *MockCustomerStore) UpdateCustomer(customer *models.Customer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.customers[customer.ID]; !exists {
		return nil // silently succeed like DynamoDB PutItem
	}
	c := *customer
	m.customers[customer.ID] = &c
	return nil
}

func (m *MockCustomerStore) DeleteCustomer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.customers, id)
	return nil
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

// MockFormStore is an in-memory implementation of FormStore for testing.
type MockFormStore struct {
	mu    sync.RWMutex
	forms map[string]*models.Form
}

func NewMockFormStore() *MockFormStore {
	return &MockFormStore{forms: make(map[string]*models.Form)}
}

func (m *MockFormStore) CreateForm(form *models.Form) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.forms[form.ID]; exists {
		return ErrAlreadyExists
	}
	f := *form
	m.forms[form.ID] = &f
	return nil
}

func (m *MockFormStore) GetForm(id string) (*models.Form, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if f, ok := m.forms[id]; ok {
		copy := *f
		return &copy, nil
	}
	return nil, nil
}

func (m *MockFormStore) ListForms(userId string) ([]models.Form, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]models.Form, 0)
	for _, f := range m.forms {
		if f.UserID == userId {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *MockFormStore) UpdateForm(form *models.Form) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f := *form
	m.forms[form.ID] = &f
	return nil
}

func (m *MockFormStore) DeleteForm(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.forms, id)
	return nil
}

// MockFieldStore is an in-memory implementation of FieldStore for testing.
type MockFieldStore struct {
	mu         sync.RWMutex
	placements map[string][]models.FieldPlacement // keyed by formId
}

func NewMockFieldStore() *MockFieldStore {
	return &MockFieldStore{placements: make(map[string][]models.FieldPlacement)}
}

func (m *MockFieldStore) SaveFieldPlacement(placement *models.FieldPlacement) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing := m.placements[placement.FormID]
	maxVersion := 0
	for _, p := range existing {
		if p.Version > maxVersion {
			maxVersion = p.Version
		}
	}

	newVersion := maxVersion + 1
	p := *placement
	p.Version = newVersion
	m.placements[placement.FormID] = append(existing, p)
	return newVersion, nil
}

func (m *MockFieldStore) GetLatestFieldPlacement(formId string) (*models.FieldPlacement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions := m.placements[formId]
	if len(versions) == 0 {
		return nil, nil
	}

	latest := versions[0]
	for _, v := range versions {
		if v.Version > latest.Version {
			latest = v
		}
	}
	copy := latest
	return &copy, nil
}

func (m *MockFieldStore) GetFieldPlacement(formId string, version int) (*models.FieldPlacement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.placements[formId] {
		if p.Version == version {
			copy := p
			return &copy, nil
		}
	}
	return nil, nil
}

func (m *MockFieldStore) ListVersions(formId string) ([]models.FieldPlacement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions := m.placements[formId]
	result := make([]models.FieldPlacement, len(versions))
	for i, v := range versions {
		result[i] = models.FieldPlacement{
			FormID:    v.FormID,
			Version:   v.Version,
			CreatedAt: v.CreatedAt,
			Source:    v.Source,
		}
	}

	// Sort by version descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version > result[j].Version
	})
	return result, nil
}

func (m *MockFieldStore) DeleteAllVersions(formId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.placements, formId)
	return nil
}
