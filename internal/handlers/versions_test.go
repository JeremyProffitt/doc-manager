package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

func setupVersionsTestApp(t *testing.T) (*fiber.App, *mockFieldStore) {
	t.Helper()
	app := fiber.New()

	flds := newMockFieldStore()
	h := NewVersionsHandler(flds)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userEmail", "test@example.com")
		return c.Next()
	})

	app.Get("/api/forms/:id/fields", h.GetCurrentFields)
	app.Get("/api/forms/:id/fields/versions", h.ListVersions)
	app.Get("/api/forms/:id/fields/:version", h.GetVersion)
	app.Post("/api/forms/:id/fields/revert/:v", h.RevertToVersion)
	app.Put("/api/forms/:id/fields", h.SaveFields)

	return app, flds
}

func TestVersionsHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		setupFields    func(flds *mockFieldStore)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:   "GET /api/forms/:id/fields returns current field placement JSON",
			method: http.MethodGet,
			path:   "/api/forms/form-1/fields",
			setupFields: func(flds *mockFieldStore) {
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:     "form-1",
					Source:     "ai_analysis",
					FontFamily: "Arial",
					FontSize:   12,
					Fields: []models.Field{
						{FieldName: "Name", Page: 1, X: 10, Y: 20, Width: 30, Height: 5, Confidence: 0.95},
					},
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp models.FieldPlacement
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp.Version != 1 {
					t.Errorf("expected version 1, got %d", resp.Version)
				}
				if len(resp.Fields) != 1 {
					t.Errorf("expected 1 field, got %d", len(resp.Fields))
				}
				if resp.Fields[0].FieldName != "Name" {
					t.Errorf("expected field name 'Name', got %q", resp.Fields[0].FieldName)
				}
			},
		},
		{
			name:           "GET /api/forms/:id/fields returns 404 when no placements exist",
			method:         http.MethodGet,
			path:           "/api/forms/no-fields/fields",
			expectedStatus: 404,
		},
		{
			name:   "GET /api/forms/:id/fields/versions returns version list",
			method: http.MethodGet,
			path:   "/api/forms/form-1/fields/versions",
			setupFields: func(flds *mockFieldStore) {
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:    "form-1",
					Source:    "ai_analysis",
					CreatedAt: "2024-01-01T00:00:00Z",
					Fields:    []models.Field{{FieldName: "Name", Page: 1, X: 10, Y: 20, Width: 30, Height: 5}},
				})
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:    "form-1",
					Source:    "manual_edit",
					CreatedAt: "2024-01-02T00:00:00Z",
					Fields:    []models.Field{{FieldName: "Name", Page: 1, X: 11, Y: 21, Width: 31, Height: 6}},
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var versions []models.FieldPlacement
				if err := json.Unmarshal([]byte(body), &versions); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if len(versions) != 2 {
					t.Errorf("expected 2 versions, got %d", len(versions))
				}
				// Should be sorted descending
				if versions[0].Version < versions[1].Version {
					t.Error("expected versions sorted descending")
				}
			},
		},
		{
			name:   "GET /api/forms/:id/fields/:version returns specific version",
			method: http.MethodGet,
			path:   "/api/forms/form-1/fields/1",
			setupFields: func(flds *mockFieldStore) {
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID: "form-1",
					Source: "ai_analysis",
					Fields: []models.Field{{FieldName: "Name", Page: 1, X: 10, Y: 20, Width: 30, Height: 5}},
				})
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID: "form-1",
					Source: "manual_edit",
					Fields: []models.Field{{FieldName: "Name", Page: 1, X: 15, Y: 25, Width: 35, Height: 6}},
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp models.FieldPlacement
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp.Version != 1 {
					t.Errorf("expected version 1, got %d", resp.Version)
				}
				if resp.Source != "ai_analysis" {
					t.Errorf("expected source 'ai_analysis', got %q", resp.Source)
				}
			},
		},
		{
			name:           "GET /api/forms/:id/fields/:version returns 404 for nonexistent version",
			method:         http.MethodGet,
			path:           "/api/forms/form-1/fields/99",
			expectedStatus: 404,
		},
		{
			name:           "GET /api/forms/:id/fields/:version returns 400 for invalid version",
			method:         http.MethodGet,
			path:           "/api/forms/form-1/fields/abc",
			expectedStatus: 400,
		},
		{
			name:   "POST /api/forms/:id/fields/revert/:v creates new version and returns it",
			method: http.MethodPost,
			path:   "/api/forms/form-1/fields/revert/1",
			setupFields: func(flds *mockFieldStore) {
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:     "form-1",
					Source:     "ai_analysis",
					FontFamily: "Arial",
					FontSize:   12,
					Fields:     []models.Field{{FieldName: "Name", Page: 1, X: 10, Y: 20, Width: 30, Height: 5}},
				})
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:     "form-1",
					Source:     "manual_edit",
					FontFamily: "Courier",
					FontSize:   14,
					Fields:     []models.Field{{FieldName: "Name", Page: 1, X: 15, Y: 25, Width: 35, Height: 6}},
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp models.FieldPlacement
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp.Version != 3 {
					t.Errorf("expected version 3 (new version after revert), got %d", resp.Version)
				}
				if resp.Source != "revert_from_v1" {
					t.Errorf("expected source 'revert_from_v1', got %q", resp.Source)
				}
				if resp.FontFamily != "Arial" {
					t.Errorf("expected font family 'Arial' from reverted version, got %q", resp.FontFamily)
				}
				if len(resp.Fields) != 1 {
					t.Errorf("expected 1 field from reverted version, got %d", len(resp.Fields))
				}
			},
		},
		{
			name:           "POST /api/forms/:id/fields/revert/:v returns 404 for nonexistent version",
			method:         http.MethodPost,
			path:           "/api/forms/form-1/fields/revert/99",
			expectedStatus: 404,
		},
		{
			name:           "POST /api/forms/:id/fields/revert/:v returns 400 for invalid version",
			method:         http.MethodPost,
			path:           "/api/forms/form-1/fields/revert/abc",
			expectedStatus: 400,
		},
		{
			name:        "PUT /api/forms/:id/fields saves new version",
			method:      http.MethodPut,
			path:        "/api/forms/form-1/fields",
			contentType: "application/json",
			body: `{
				"fontFamily": "Helvetica",
				"fontSize": 11,
				"fields": [
					{"fieldName": "Name", "page": 1, "x": 12, "y": 22, "width": 32, "height": 6},
					{"fieldName": "Date", "page": 1, "x": 60, "y": 22, "width": 20, "height": 6}
				]
			}`,
			setupFields: func(flds *mockFieldStore) {
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID: "form-1",
					Source: "ai_analysis",
					Fields: []models.Field{{FieldName: "Name", Page: 1, X: 10, Y: 20, Width: 30, Height: 5}},
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp models.FieldPlacement
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp.Version != 2 {
					t.Errorf("expected version 2, got %d", resp.Version)
				}
				if resp.Source != "manual_edit" {
					t.Errorf("expected source 'manual_edit', got %q", resp.Source)
				}
				if resp.FontFamily != "Helvetica" {
					t.Errorf("expected font family 'Helvetica', got %q", resp.FontFamily)
				}
				if len(resp.Fields) != 2 {
					t.Errorf("expected 2 fields, got %d", len(resp.Fields))
				}
			},
		},
		{
			name:           "PUT /api/forms/:id/fields returns 400 for invalid body",
			method:         http.MethodPut,
			path:           "/api/forms/form-1/fields",
			contentType:    "application/json",
			body:           `{invalid json`,
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, flds := setupVersionsTestApp(t)

			if tt.setupFields != nil {
				tt.setupFields(flds)
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
