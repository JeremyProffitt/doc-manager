package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func TestHome(t *testing.T) {
	engine := html.New("../../templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userEmail", "test@example.com")
		return c.Next()
	})

	app.Get("/", Home)

	tests := []struct {
		name           string
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:           "GET / returns 200 with dashboard",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Dashboard") {
					t.Error("expected home page to contain 'Dashboard'")
				}
				if !strings.Contains(body, "test@example.com") {
					t.Error("expected home page to contain user email")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}

			if tt.checkBody != nil {
				body, _ := io.ReadAll(resp.Body)
				tt.checkBody(t, string(body))
			}
		})
	}
}
