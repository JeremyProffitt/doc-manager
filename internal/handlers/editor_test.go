package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

func setupEditorTestApp(t *testing.T) (*fiber.App, *mockFormStore, *mockFieldStore, *mockS3Service) {
	t.Helper()
	engine := html.New("../../templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	fs := newMockFormStore()
	flds := newMockFieldStore()
	s3svc := newMockS3Service()

	h := NewEditorHandler(fs, flds, s3svc)

	// Simulate auth middleware setting userEmail
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userEmail", "test@example.com")
		return c.Next()
	})

	app.Get("/forms/:id/edit", h.EditForm)

	return app, fs, flds, s3svc
}

func TestEditorHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setupData      func(fs *mockFormStore, flds *mockFieldStore)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name: "GET /forms/:id/edit returns 200 with form data and field placements",
			path: "/forms/form-1/edit",
			setupData: func(fs *mockFormStore, flds *mockFieldStore) {
				fs.CreateForm(&models.Form{
					ID:         "form-1",
					UserID:     "test@example.com",
					Name:       "W-9 Test Form",
					Status:     "analyzed",
					S3Key:      "forms/form-1/w9.pdf",
					FontFamily: "Helvetica",
					FontSize:   12,
				})
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:     "form-1",
					Source:     "ai_analysis",
					FontFamily: "Helvetica",
					FontSize:   12,
					Fields: []models.Field{
						{
							FieldName:  "Name",
							Page:       1,
							X:          15.2,
							Y:          13.5,
							Width:      50.0,
							Height:     3.2,
							Confidence: 0.95,
						},
						{
							FieldName:  "Address",
							Page:       1,
							X:          15.2,
							Y:          35.0,
							Width:      50.0,
							Height:     3.2,
							Confidence: 0.88,
						},
					},
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				// Should contain the form name
				if !strings.Contains(body, "W-9 Test Form") {
					t.Error("expected editor page to contain form name 'W-9 Test Form'")
				}
				// Should contain field data as JSON (field names appear in serialized JSON)
				if !strings.Contains(body, "Name") {
					t.Error("expected editor page to contain field name 'Name'")
				}
				if !strings.Contains(body, "Address") {
					t.Error("expected editor page to contain field name 'Address'")
				}
				// Should contain the form ID for JavaScript
				if !strings.Contains(body, "form-1") {
					t.Error("expected editor page to contain form ID")
				}
			},
		},
		{
			name:           "GET /forms/:id/edit for non-existent form returns 404",
			path:           "/forms/no-such-form/edit",
			expectedStatus: 404,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Form not found") {
					t.Error("expected 404 body to contain 'Form not found'")
				}
			},
		},
		{
			name: "GET /forms/:id/edit includes font settings in response",
			path: "/forms/form-2/edit",
			setupData: func(fs *mockFormStore, flds *mockFieldStore) {
				fs.CreateForm(&models.Form{
					ID:         "form-2",
					UserID:     "test@example.com",
					Name:       "Tax Form",
					Status:     "uploaded",
					S3Key:      "forms/form-2/tax.pdf",
					FontFamily: "Courier",
					FontSize:   10,
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				// Should contain font family from form
				if !strings.Contains(body, "Courier") {
					t.Error("expected editor page to contain font family 'Courier'")
				}
				// Should contain the form name
				if !strings.Contains(body, "Tax Form") {
					t.Error("expected editor page to contain form name 'Tax Form'")
				}
			},
		},
		{
			name: "GET /forms/:id/edit with no field placements returns 200",
			path: "/forms/form-3/edit",
			setupData: func(fs *mockFormStore, flds *mockFieldStore) {
				fs.CreateForm(&models.Form{
					ID:         "form-3",
					UserID:     "test@example.com",
					Name:       "Empty Form",
					Status:     "uploaded",
					S3Key:      "forms/form-3/empty.pdf",
					FontFamily: "Helvetica",
					FontSize:   12,
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Empty Form") {
					t.Error("expected editor page to contain form name 'Empty Form'")
				}
				// Should still render with null/empty placement data
				if !strings.Contains(body, "initEditor") {
					t.Error("expected editor page to contain initEditor call")
				}
			},
		},
		{
			name: "GET /forms/:id/edit includes version history data",
			path: "/forms/form-4/edit",
			setupData: func(fs *mockFormStore, flds *mockFieldStore) {
				fs.CreateForm(&models.Form{
					ID:         "form-4",
					UserID:     "test@example.com",
					Name:       "Versioned Form",
					Status:     "analyzed",
					S3Key:      "forms/form-4/versioned.pdf",
					FontFamily: "Helvetica",
					FontSize:   12,
				})
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:     "form-4",
					Source:     "ai_analysis",
					FontFamily: "Helvetica",
					FontSize:   12,
					Fields: []models.Field{
						{FieldName: "Name", Page: 1, X: 10, Y: 10, Width: 20, Height: 3, Confidence: 0.9},
					},
				})
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:     "form-4",
					Source:     "manual_edit",
					FontFamily: "Helvetica",
					FontSize:   12,
					Fields: []models.Field{
						{FieldName: "Name", Page: 1, X: 12, Y: 12, Width: 22, Height: 3, Confidence: 1.0},
					},
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Versioned Form") {
					t.Error("expected editor page to contain form name 'Versioned Form'")
				}
				// Version history JSON should contain source types
				if !strings.Contains(body, "ai_analysis") {
					t.Error("expected editor page to contain version source 'ai_analysis'")
				}
				if !strings.Contains(body, "manual_edit") {
					t.Error("expected editor page to contain version source 'manual_edit'")
				}
			},
		},
		{
			name: "GET /forms/:id/edit includes image URL from S3",
			path: "/forms/form-5/edit",
			setupData: func(fs *mockFormStore, flds *mockFieldStore) {
				fs.CreateForm(&models.Form{
					ID:         "form-5",
					UserID:     "test@example.com",
					Name:       "Image Form",
					Status:     "uploaded",
					S3Key:      "forms/form-5/image.pdf",
					FontFamily: "Times-Roman",
					FontSize:   14,
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				// The mock S3 service returns a URL containing the s3Key.
				// Go html/template JS-escapes forward slashes inside <script> tags,
				// so check for the escaped form as well as the raw form.
				hasRaw := strings.Contains(body, "forms/form-5/image.pdf")
				hasEscaped := strings.Contains(body, `forms\/form-5\/image.pdf`)
				if !hasRaw && !hasEscaped {
					t.Error("expected editor page to contain S3 image URL")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, fs, flds, _ := setupEditorTestApp(t)

			if tt.setupData != nil {
				tt.setupData(fs, flds)
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
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
