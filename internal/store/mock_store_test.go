package store

import (
	"errors"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// ErrAlreadyExists is returned when an item already exists in the store.
var ErrAlreadyExists = errors.New("item already exists")

// mockUserStore is an in-memory implementation of UserStore for testing.
type mockUserStore struct {
	users map[string]*models.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]*models.User)}
}

func (m *mockUserStore) CreateUser(user *models.User) error {
	if _, exists := m.users[user.Email]; exists {
		return ErrAlreadyExists
	}
	u := *user
	m.users[user.Email] = &u
	return nil
}

func (m *mockUserStore) GetUser(email string) (*models.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, nil
}

// mockCustomerStore is an in-memory implementation of CustomerStore for testing.
type mockCustomerStore struct {
	customers map[string]*models.Customer
}

func newMockCustomerStore() *mockCustomerStore {
	return &mockCustomerStore{customers: make(map[string]*models.Customer)}
}

func (m *mockCustomerStore) CreateCustomer(customer *models.Customer) error {
	if _, exists := m.customers[customer.ID]; exists {
		return ErrAlreadyExists
	}
	c := *customer
	m.customers[customer.ID] = &c
	return nil
}

func (m *mockCustomerStore) GetCustomer(id string) (*models.Customer, error) {
	if c, ok := m.customers[id]; ok {
		return c, nil
	}
	return nil, nil
}

func (m *mockCustomerStore) ListCustomers() ([]models.Customer, error) {
	result := make([]models.Customer, 0, len(m.customers))
	for _, c := range m.customers {
		result = append(result, *c)
	}
	return result, nil
}

// mockSettingsStore is an in-memory implementation of SettingsStore for testing.
type mockSettingsStore struct {
	settings map[string]*models.Setting
}

func newMockSettingsStore() *mockSettingsStore {
	return &mockSettingsStore{settings: make(map[string]*models.Setting)}
}

func (m *mockSettingsStore) GetSetting(key string) (*models.Setting, error) {
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockSettingsStore) PutSetting(setting *models.Setting) error {
	if _, exists := m.settings[setting.Key]; exists {
		return ErrAlreadyExists
	}
	s := *setting
	m.settings[setting.Key] = &s
	return nil
}
