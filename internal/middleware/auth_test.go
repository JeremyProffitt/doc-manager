package middleware

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

const testJWTSecret = "test-secret-key-for-middleware"

// mockSessionStore is a test-local mock for store.SessionStore.
type mockSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*models.Session
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]*models.Session)}
}

func (m *mockSessionStore) CreateSession(session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := *session
	m.sessions[session.Token] = &c
	return nil
}

func (m *mockSessionStore) GetSession(token string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[token]
	if !ok {
		return nil, nil
	}
	if s.ExpiresAt < time.Now().Unix() {
		return nil, nil
	}
	c := *s
	return &c, nil
}

func (m *mockSessionStore) DeleteSession(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
	return nil
}

// Compile-time check that mockSessionStore satisfies store.SessionStore.
var _ store.SessionStore = (*mockSessionStore)(nil)

// createValidJWT creates a valid JWT with the given session token.
func createValidJWT(sessionToken string, expiresAt time.Time) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_token": sessionToken,
		"exp":           expiresAt.Unix(),
	})
	signed, _ := token.SignedString([]byte(testJWTSecret))
	return signed
}

// setupTestApp creates a Fiber app with auth middleware and a protected route.
func setupTestApp(ss store.SessionStore) *fiber.App {
	app := fiber.New()
	app.Use(AuthRequired(ss, testJWTSecret))

	// Protected route that echoes back the user email from context
	app.Get("/", func(c *fiber.Ctx) error {
		email := c.Locals("userEmail")
		return c.SendString(fmt.Sprintf("hello %v", email))
	})

	// Login page (exempt from auth)
	app.Get("/login", func(c *fiber.Ctx) error {
		return c.SendString("login page")
	})

	// Health check (exempt from auth)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Static file (exempt from auth)
	app.Get("/static/css/app.css", func(c *fiber.Ctx) error {
		return c.SendString("body{}")
	})

	return app
}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setupSession   func(ss *mockSessionStore) string // returns cookie value
		expectedStatus int
		expectedBody   string
		checkRedirect  bool
		redirectTarget string
	}{
		{
			name: "valid session cookie passes through with user email in context",
			path: "/",
			setupSession: func(ss *mockSessionStore) string {
				ss.CreateSession(&models.Session{
					Token:     "valid-token-123",
					UserEmail: "alice@example.com",
					ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
				})
				return createValidJWT("valid-token-123", time.Now().Add(24*time.Hour))
			},
			expectedStatus: 200,
			expectedBody:   "hello alice@example.com",
		},
		{
			name: "no cookie redirects to login",
			path: "/",
			setupSession: func(ss *mockSessionStore) string {
				return "" // no cookie
			},
			expectedStatus: 302,
			checkRedirect:  true,
			redirectTarget: "/login",
		},
		{
			name: "invalid JWT redirects to login",
			path: "/",
			setupSession: func(ss *mockSessionStore) string {
				return "not-a-valid-jwt"
			},
			expectedStatus: 302,
			checkRedirect:  true,
			redirectTarget: "/login",
		},
		{
			name: "expired session redirects to login",
			path: "/",
			setupSession: func(ss *mockSessionStore) string {
				ss.CreateSession(&models.Session{
					Token:     "expired-token",
					UserEmail: "alice@example.com",
					ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
				})
				// JWT itself is still valid but session in store is expired
				return createValidJWT("expired-token", time.Now().Add(24*time.Hour))
			},
			expectedStatus: 302,
			checkRedirect:  true,
			redirectTarget: "/login",
		},
		{
			name: "login path is exempt from auth check",
			path: "/login",
			setupSession: func(ss *mockSessionStore) string {
				return "" // no cookie
			},
			expectedStatus: 200,
			expectedBody:   "login page",
		},
		{
			name: "static path is exempt from auth check",
			path: "/static/css/app.css",
			setupSession: func(ss *mockSessionStore) string {
				return "" // no cookie
			},
			expectedStatus: 200,
			expectedBody:   "body{}",
		},
		{
			name: "health path is exempt from auth check",
			path: "/health",
			setupSession: func(ss *mockSessionStore) string {
				return "" // no cookie
			},
			expectedStatus: 200,
			expectedBody:   "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := newMockSessionStore()
			cookie := tt.setupSession(ss)
			app := setupTestApp(ss)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if cookie != "" {
				req.AddCookie(&http.Cookie{Name: "session", Value: cookie})
			}

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkRedirect {
				loc := resp.Header.Get("Location")
				if loc != tt.redirectTarget {
					t.Errorf("expected redirect to %s, got %s", tt.redirectTarget, loc)
				}
			}

			if tt.expectedBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if string(body) != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, string(body))
				}
			}
		})
	}
}
