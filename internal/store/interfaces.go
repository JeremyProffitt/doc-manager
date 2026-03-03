package store

import (
	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// UserStore defines the interface for user persistence operations.
type UserStore interface {
	GetUser(email string) (*models.User, error)
	CreateUser(user *models.User) error
}

// SessionStore defines the interface for session persistence operations.
type SessionStore interface {
	CreateSession(session *models.Session) error
	GetSession(token string) (*models.Session, error)
	DeleteSession(token string) error
}

// CustomerStore defines operations for managing customers in the data store.
type CustomerStore interface {
	CreateCustomer(customer *models.Customer) error
	GetCustomer(id string) (*models.Customer, error)
	ListCustomers() ([]models.Customer, error)
	UpdateCustomer(customer *models.Customer) error
	DeleteCustomer(id string) error
}

// SettingsStore defines operations for managing settings in the data store.
type SettingsStore interface {
	GetSetting(key string) (*models.Setting, error)
	PutSetting(setting *models.Setting) error
}

// FormStore defines operations for managing forms in the data store.
type FormStore interface {
	CreateForm(form *models.Form) error
	GetForm(id string) (*models.Form, error)
	ListForms(userId string) ([]models.Form, error)
	UpdateForm(form *models.Form) error
	DeleteForm(id string) error
}

// FieldStore defines operations for managing versioned field placements.
type FieldStore interface {
	SaveFieldPlacement(placement *models.FieldPlacement) (int, error) // returns new version number
	GetLatestFieldPlacement(formId string) (*models.FieldPlacement, error)
	GetFieldPlacement(formId string, version int) (*models.FieldPlacement, error)
	ListVersions(formId string) ([]models.FieldPlacement, error) // metadata only (no fields array)
	DeleteAllVersions(formId string) error
}
