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
