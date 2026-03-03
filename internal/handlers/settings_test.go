package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

func setupSettingsTestApp(t *testing.T) (*fiber.App, *mockSettingsStore) {
	t.Helper()
	engine := html.New("../../templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	ss := newMockSettingsStore()
	h := NewSettingsHandler(ss)

	// Simulate auth middleware setting userEmail
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userEmail", "test@example.com")
		return c.Next()
	})

	app.Get("/settings/fields", h.GetFields)
	app.Put("/api/settings/fields", h.UpdateFields)

	return app, ss
}

func TestSettingsHandlers(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		setupSettings  func(ss *mockSettingsStore)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:   "GET /settings/fields returns 200 with HTML",
			method: http.MethodGet,
			path:   "/settings/fields",
			setupSettings: func(ss *mockSettingsStore) {
				fields := []models.FieldDefinition{
					{Name: "Name", Type: "text"},
					{Name: "Address", Type: "text"},
				}
				value, _ := json.Marshal(fields)
				ss.PutSetting(&models.Setting{
					Key:   "standard_fields",
					Value: string(value),
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Name") {
					t.Error("expected settings page to contain field 'Name'")
				}
			},
		},
		{
			name:           "GET /settings/fields with no settings returns 200",
			method:         http.MethodGet,
			path:           "/settings/fields",
			expectedStatus: 200,
		},
		{
			name:        "PUT /api/settings/fields updates settings",
			method:      http.MethodPut,
			path:        "/api/settings/fields",
			body:        `[{"name":"Name","type":"text"},{"name":"Phone","type":"text"}]`,
			contentType: "application/json",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp map[string]string
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp["status"] != "ok" {
					t.Errorf("expected status 'ok', got %s", resp["status"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ss := setupSettingsTestApp(t)

			if tt.setupSettings != nil {
				tt.setupSettings(ss)
			}

			var reqBody io.Reader
			if tt.body != "" {
				reqBody = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, reqBody)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

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
