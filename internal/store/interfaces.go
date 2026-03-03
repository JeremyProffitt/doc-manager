package store

import (
	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// UserStore defines operations for managing users in the data store.
type UserStore interface {
	CreateUser(user *models.User) error
	GetUser(email string) (*models.User, error)
}

// CustomerStore defines operations for managing customers in the data store.
type CustomerStore interface {
	CreateCustomer(customer *models.Customer) error
	GetCustomer(id string) (*models.Customer, error)
	ListCustomers() ([]models.Customer, error)
}

// SettingsStore defines operations for managing settings in the data store.
type SettingsStore interface {
	GetSetting(key string) (*models.Setting, error)
	PutSetting(setting *models.Setting) error
}
