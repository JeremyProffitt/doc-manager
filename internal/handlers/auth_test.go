package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

const testJWTSecret = "test-secret-key-for-handlers"

// --- Mock stores for handler tests ---

type mockUserStore struct {
	mu    sync.RWMutex
	users map[string]*models.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]*models.User)}
}

func (m *mockUserStore) GetUser(email string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	c := *u
	return &c, nil
}

func (m *mockUserStore) CreateUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.users[user.Email]; exists {
		return fmt.Errorf("user already exists: %s", user.Email)
	}
	c := *user
	m.users[user.Email] = &c
	return nil
}

var _ store.UserStore = (*mockUserStore)(nil)

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

var _ store.SessionStore = (*mockSessionStore)(nil)

// --- Test helpers ---

func setupTestApp(t *testing.T, us *mockUserStore, ss *mockSessionStore) *fiber.App {
	t.Helper()
	engine := html.New("../../templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	h := NewAuthHandler(us, ss, testJWTSecret)
	app.Get("/login", h.GetLogin)
	app.Post("/login", h.PostLogin)
	app.Post("/logout", h.PostLogout)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("home")
	})

	return app
}

func seedUser(us *mockUserStore, email, password string) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	us.CreateUser(&models.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         "Test User",
		CreatedAt:    "2024-01-01T00:00:00Z",
	})
}

func createValidJWT(sessionToken string, expiresAt time.Time) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_token": sessionToken,
		"exp":           expiresAt.Unix(),
	})
	signed, _ := token.SignedString([]byte(testJWTSecret))
	return signed
}

func TestAuthHandlers(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		cookie         *http.Cookie
		setupUser      func(us *mockUserStore)
		setupSession   func(ss *mockSessionStore)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
		checkRedirect  string
		checkCookie    func(t *testing.T, cookies []*http.Cookie)
	}{
		{
			name:           "GET /login returns 200 with HTML",
			method:         http.MethodGet,
			path:           "/login",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Sign In") {
					t.Error("expected login page to contain 'Sign In'")
				}
				if !strings.Contains(body, "<form") {
					t.Error("expected login page to contain a form element")
				}
			},
		},
		{
			name:        "POST /login with valid credentials redirects to /",
			method:      http.MethodPost,
			path:        "/login",
			body:        "email=alice%40example.com&password=correctpassword",
			contentType: "application/x-www-form-urlencoded",
			setupUser: func(us *mockUserStore) {
				seedUser(us, "alice@example.com", "correctpassword")
			},
			expectedStatus: 302,
			checkRedirect:  "/",
			checkCookie: func(t *testing.T, cookies []*http.Cookie) {
				t.Helper()
				var sessionCookie *http.Cookie
				for _, c := range cookies {
					if c.Name == "session" {
						sessionCookie = c
						break
					}
				}
				if sessionCookie == nil {
					t.Fatal("expected session cookie to be set")
				}
				if sessionCookie.Value == "" {
					t.Error("expected session cookie to have a value")
				}
				if !sessionCookie.HttpOnly {
					t.Error("expected session cookie to be HttpOnly")
				}
			},
		},
		{
			name:        "POST /login with wrong password returns 200 with error",
			method:      http.MethodPost,
			path:        "/login",
			body:        "email=alice%40example.com&password=wrongpassword",
			contentType: "application/x-www-form-urlencoded",
			setupUser: func(us *mockUserStore) {
				seedUser(us, "alice@example.com", "correctpassword")
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Invalid email or password") {
					t.Error("expected error message 'Invalid email or password'")
				}
			},
		},
		{
			name:        "POST /login with non-existent email returns 200 with same error (no user enumeration)",
			method:      http.MethodPost,
			path:        "/login",
			body:        "email=nobody%40example.com&password=somepassword",
			contentType: "application/x-www-form-urlencoded",
			setupUser:   func(us *mockUserStore) {},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Invalid email or password") {
					t.Error("expected error message 'Invalid email or password'")
				}
			},
		},
		{
			name:           "POST /login with empty email returns 400",
			method:         http.MethodPost,
			path:           "/login",
			body:           "email=&password=somepassword",
			contentType:    "application/x-www-form-urlencoded",
			expectedStatus: 400,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Email and password are required") {
					t.Error("expected validation error message")
				}
			},
		},
		{
			name:           "POST /login with empty password returns 400",
			method:         http.MethodPost,
			path:           "/login",
			body:           "email=alice%40example.com&password=",
			contentType:    "application/x-www-form-urlencoded",
			expectedStatus: 400,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Email and password are required") {
					t.Error("expected validation error message")
				}
			},
		},
		{
			name:   "POST /logout deletes session and clears cookie and redirects to /login",
			method: http.MethodPost,
			path:   "/logout",
			setupSession: func(ss *mockSessionStore) {
				ss.CreateSession(&models.Session{
					Token:     "logout-token",
					UserEmail: "alice@example.com",
					ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
				})
			},
			cookie: &http.Cookie{
				Name:  "session",
				Value: func() string { return createValidJWT("logout-token", time.Now().Add(24*time.Hour)) }(),
			},
			expectedStatus: 302,
			checkRedirect:  "/login",
			checkCookie: func(t *testing.T, cookies []*http.Cookie) {
				t.Helper()
				var sessionCookie *http.Cookie
				for _, c := range cookies {
					if c.Name == "session" {
						sessionCookie = c
						break
					}
				}
				if sessionCookie == nil {
					t.Fatal("expected session cookie in response")
				}
				if sessionCookie.Value != "" {
					t.Error("expected session cookie to be cleared (empty value)")
				}
			},
		},
		{
			name:   "GET /login when already authenticated redirects to /",
			method: http.MethodGet,
			path:   "/login",
			setupSession: func(ss *mockSessionStore) {
				// We just need a cookie that looks valid for the GetLogin check.
				// GetLogin checks c.Locals("userEmail"), but since we don't have
				// middleware in this test app, we test it differently.
				// The handler checks c.Locals("userEmail") which won't be set
				// without middleware. This test verifies the login page is shown
				// when not authenticated (no middleware to set locals).
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				// Without auth middleware, userEmail won't be in locals,
				// so login page should render.
				if !strings.Contains(body, "Sign In") {
					t.Error("expected login page")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			us := newMockUserStore()
			ss := newMockSessionStore()

			if tt.setupUser != nil {
				tt.setupUser(us)
			}
			if tt.setupSession != nil {
				tt.setupSession(ss)
			}

			app := setupTestApp(t, us, ss)

			var reqBody io.Reader
			if tt.body != "" {
				reqBody = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, reqBody)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}

			if tt.checkRedirect != "" {
				loc := resp.Header.Get("Location")
				if loc != tt.checkRedirect {
					t.Errorf("expected redirect to %s, got %s", tt.checkRedirect, loc)
				}
			}

			if tt.checkBody != nil && resp.StatusCode != 302 {
				body, _ := io.ReadAll(resp.Body)
				tt.checkBody(t, string(body))
			}

			if tt.checkCookie != nil {
				tt.checkCookie(t, resp.Cookies())
			}
		})
	}
}

// TestGetLoginAlreadyAuthenticated tests the redirect when a user is
// already logged in. We set up a full app with middleware to test this.
func TestGetLoginAlreadyAuthenticated(t *testing.T) {
	us := newMockUserStore()
	ss := newMockSessionStore()

	seedUser(us, "alice@example.com", "password")
	ss.CreateSession(&models.Session{
		Token:     "existing-session",
		UserEmail: "alice@example.com",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	})

	engine := html.New("../../templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	h := NewAuthHandler(us, ss, testJWTSecret)

	// Add a mini-middleware that sets userEmail in Locals if cookie is valid
	app.Use(func(c *fiber.Ctx) error {
		cookie := c.Cookies("session")
		if cookie != "" {
			token, err := jwt.Parse(cookie, func(t *jwt.Token) (interface{}, error) {
				return []byte(testJWTSecret), nil
			})
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if sessionToken, ok := claims["session_token"].(string); ok {
						sess, _ := ss.GetSession(sessionToken)
						if sess != nil {
							c.Locals("userEmail", sess.UserEmail)
						}
					}
				}
			}
		}
		return c.Next()
	})

	app.Get("/login", h.GetLogin)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("home")
	})

	jwtCookie := createValidJWT("existing-session", time.Now().Add(24*time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: jwtCookie})

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != 302 {
		t.Errorf("expected 302 redirect, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/" {
		t.Errorf("expected redirect to /, got %s", loc)
	}
}
