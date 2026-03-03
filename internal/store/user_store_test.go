package store

import (
	"testing"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

func TestDynamoUserStore(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(s UserStore)
		action  func(s UserStore) (interface{}, error)
		check   func(t *testing.T, result interface{}, err error)
	}{
		{
			name: "GetUser with existing email returns user",
			setup: func(s UserStore) {
				s.CreateUser(&models.User{
					Email:        "alice@example.com",
					PasswordHash: "$2a$10$abcdef",
					Name:         "Alice",
					CreatedAt:    "2024-01-01T00:00:00Z",
				})
			},
			action: func(s UserStore) (interface{}, error) {
				return s.GetUser("alice@example.com")
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				user, ok := result.(*models.User)
				if !ok || user == nil {
					t.Fatal("expected user, got nil")
				}
				if user.Email != "alice@example.com" {
					t.Errorf("expected email alice@example.com, got %s", user.Email)
				}
				if user.Name != "Alice" {
					t.Errorf("expected name Alice, got %s", user.Name)
				}
			},
		},
		{
			name:  "GetUser with non-existent email returns nil nil",
			setup: func(s UserStore) {},
			action: func(s UserStore) (interface{}, error) {
				return s.GetUser("nobody@example.com")
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.(*models.User) != nil {
					t.Fatal("expected nil user for non-existent email")
				}
			},
		},
		{
			name:  "CreateUser succeeds",
			setup: func(s UserStore) {},
			action: func(s UserStore) (interface{}, error) {
				err := s.CreateUser(&models.User{
					Email:        "bob@example.com",
					PasswordHash: "$2a$10$xyz",
					Name:         "Bob",
					CreatedAt:    "2024-01-01T00:00:00Z",
				})
				return nil, err
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error creating user: %v", err)
				}
			},
		},
		{
			name: "CreateUser with duplicate email returns error",
			setup: func(s UserStore) {
				s.CreateUser(&models.User{
					Email:        "dup@example.com",
					PasswordHash: "$2a$10$first",
					Name:         "First",
					CreatedAt:    "2024-01-01T00:00:00Z",
				})
			},
			action: func(s UserStore) (interface{}, error) {
				err := s.CreateUser(&models.User{
					Email:        "dup@example.com",
					PasswordHash: "$2a$10$second",
					Name:         "Second",
					CreatedAt:    "2024-01-02T00:00:00Z",
				})
				return nil, err
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("expected error for duplicate email, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockUserStore()
			tt.setup(store)
			result, err := tt.action(store)
			tt.check(t, result, err)
		})
	}
}
