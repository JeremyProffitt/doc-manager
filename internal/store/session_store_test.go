package store

import (
	"testing"
	"time"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

func TestDynamoSessionStore(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(s SessionStore)
		action func(s SessionStore) (interface{}, error)
		check  func(t *testing.T, result interface{}, err error)
	}{
		{
			name:  "CreateSession succeeds",
			setup: func(s SessionStore) {},
			action: func(s SessionStore) (interface{}, error) {
				err := s.CreateSession(&models.Session{
					Token:     "tok-abc123",
					UserEmail: "alice@example.com",
					ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
				})
				return nil, err
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error creating session: %v", err)
				}
			},
		},
		{
			name: "GetSession with valid token returns session",
			setup: func(s SessionStore) {
				s.CreateSession(&models.Session{
					Token:     "tok-valid",
					UserEmail: "alice@example.com",
					ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
				})
			},
			action: func(s SessionStore) (interface{}, error) {
				return s.GetSession("tok-valid")
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				sess, ok := result.(*models.Session)
				if !ok || sess == nil {
					t.Fatal("expected session, got nil")
				}
				if sess.Token != "tok-valid" {
					t.Errorf("expected token tok-valid, got %s", sess.Token)
				}
				if sess.UserEmail != "alice@example.com" {
					t.Errorf("expected email alice@example.com, got %s", sess.UserEmail)
				}
			},
		},
		{
			name:  "GetSession with non-existent token returns nil nil",
			setup: func(s SessionStore) {},
			action: func(s SessionStore) (interface{}, error) {
				return s.GetSession("tok-nonexistent")
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.(*models.Session) != nil {
					t.Fatal("expected nil session for non-existent token")
				}
			},
		},
		{
			name: "GetSession with expired session returns nil nil",
			setup: func(s SessionStore) {
				s.CreateSession(&models.Session{
					Token:     "tok-expired",
					UserEmail: "alice@example.com",
					ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // expired
				})
			},
			action: func(s SessionStore) (interface{}, error) {
				return s.GetSession("tok-expired")
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.(*models.Session) != nil {
					t.Fatal("expected nil session for expired token")
				}
			},
		},
		{
			name: "DeleteSession succeeds",
			setup: func(s SessionStore) {
				s.CreateSession(&models.Session{
					Token:     "tok-delete",
					UserEmail: "alice@example.com",
					ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
				})
			},
			action: func(s SessionStore) (interface{}, error) {
				err := s.DeleteSession("tok-delete")
				return nil, err
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error deleting session: %v", err)
				}
			},
		},
		{
			name:  "DeleteSession non-existent returns no error",
			setup: func(s SessionStore) {},
			action: func(s SessionStore) (interface{}, error) {
				err := s.DeleteSession("tok-doesnt-exist")
				return nil, err
			},
			check: func(t *testing.T, result interface{}, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected no error deleting non-existent session, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockSessionStore()
			tt.setup(store)
			result, err := tt.action(store)
			tt.check(t, result, err)
		})
	}
}
