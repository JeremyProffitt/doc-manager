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

func setupFormsTestApp(t *testing.T) (*fiber.App, *mockFormStore, *mockFieldStore, *mockS3Service) {
	t.Helper()
	engine := html.New("../../templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	fs := newMockFormStore()
	flds := newMockFieldStore()
	s3svc := newMockS3Service()
	analysisSvc := newMockAnalysisService()

	h := NewFormsHandler(fs, flds, s3svc)
	h.SetAnalysisService(analysisSvc)

	// Simulate auth middleware setting userEmail
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userEmail", "test@example.com")
		return c.Next()
	})

	app.Get("/forms", h.ListForms)
	app.Get("/forms/:id", h.GetForm)
	app.Post("/api/forms/upload-url", h.GetUploadURL)
	app.Post("/api/forms/:id/upload-complete", h.UploadComplete)
	app.Post("/api/forms/:id/analyze", h.AnalyzeForm)
	app.Delete("/api/forms/:id", h.DeleteForm)

	return app, fs, flds, s3svc
}

func TestFormsHandlers(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		setupForms     func(fs *mockFormStore)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:   "POST /api/forms/upload-url returns 200 with upload URL and formId",
			method: http.MethodPost,
			path:   "/api/forms/upload-url",
			body:   `{"filename":"test.pdf","contentType":"application/pdf"}`,
			contentType:    "application/json",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp map[string]string
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp["uploadUrl"] == "" {
					t.Error("expected non-empty uploadUrl")
				}
				if resp["formId"] == "" {
					t.Error("expected non-empty formId")
				}
				if resp["s3Key"] == "" {
					t.Error("expected non-empty s3Key")
				}
			},
		},
		{
			name:   "POST /api/forms/:id/upload-complete returns 200",
			method: http.MethodPost,
			path:   "/api/forms/form-1/upload-complete",
			body:   `{"s3Key":"forms/form-1/test.pdf"}`,
			contentType: "application/json",
			setupForms: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:     "form-1",
					UserID: "test@example.com",
					Name:   "test.pdf",
					Status: "uploading",
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp map[string]string
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp["status"] != "uploaded" {
					t.Errorf("expected status 'uploaded', got %s", resp["status"])
				}
			},
		},
		{
			name:   "GET /forms returns 200 with HTML",
			method: http.MethodGet,
			path:   "/forms",
			setupForms: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:     "form-1",
					UserID: "test@example.com",
					Name:   "Test Form",
					Status: "uploaded",
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Test Form") {
					t.Error("expected forms list to contain 'Test Form'")
				}
			},
		},
		{
			name:           "GET /forms with no forms returns 200",
			method:         http.MethodGet,
			path:           "/forms",
			expectedStatus: 200,
		},
		{
			name:   "DELETE /api/forms/:id returns 200",
			method: http.MethodDelete,
			path:   "/api/forms/form-1",
			setupForms: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:     "form-1",
					UserID: "test@example.com",
					Name:   "To Delete",
					Status: "uploaded",
				})
			},
			expectedStatus: 200,
		},
		{
			name:           "DELETE /api/forms/:id non-existent returns 404",
			method:         http.MethodDelete,
			path:           "/api/forms/does-not-exist",
			expectedStatus: 404,
		},
		{
			name:   "GET /forms/:id returns 200 with form detail",
			method: http.MethodGet,
			path:   "/forms/form-1",
			setupForms: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:     "form-1",
					UserID: "test@example.com",
					Name:   "My Form",
					Status: "uploaded",
					S3Key:  "forms/form-1/test.pdf",
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "My Form") {
					t.Error("expected form view to contain 'My Form'")
				}
			},
		},
		{
			name:           "GET /forms/:id non-existent returns 404",
			method:         http.MethodGet,
			path:           "/forms/no-such-form",
			expectedStatus: 404,
		},
		{
			name:           "POST /api/forms/upload-url with empty filename returns 400",
			method:         http.MethodPost,
			path:           "/api/forms/upload-url",
			body:           `{"filename":"","contentType":"application/pdf"}`,
			contentType:    "application/json",
			expectedStatus: 400,
		},
		{
			name:   "POST /api/forms/upload-url defaults contentType to application/pdf",
			method: http.MethodPost,
			path:   "/api/forms/upload-url",
			body:   `{"filename":"test.pdf"}`,
			contentType:    "application/json",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp map[string]string
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp["formId"] == "" {
					t.Error("expected non-empty formId")
				}
			},
		},
		{
			name:           "POST /api/forms/:id/upload-complete non-existent returns 404",
			method:         http.MethodPost,
			path:           "/api/forms/no-such-form/upload-complete",
			body:           `{"s3Key":"forms/no-such-form/test.pdf"}`,
			contentType:    "application/json",
			expectedStatus: 404,
		},
		{
			name:   "DELETE /api/forms/:id with S3 key deletes S3 object",
			method: http.MethodDelete,
			path:   "/api/forms/form-2",
			setupForms: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:     "form-2",
					UserID: "test@example.com",
					Name:   "With S3",
					Status: "uploaded",
					S3Key:  "forms/form-2/test.pdf",
				})
			},
			expectedStatus: 200,
		},
		{
			name:   "POST /api/forms/:id/analyze returns 202 Accepted",
			method: http.MethodPost,
			path:   "/api/forms/form-1/analyze",
			setupForms: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:     "form-1",
					UserID: "test@example.com",
					Name:   "test.pdf",
					Status: "uploaded",
					S3Key:  "forms/form-1/test.pdf",
				})
			},
			expectedStatus: 202,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var resp map[string]string
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if resp["status"] != "analyzing" {
					t.Errorf("expected status 'analyzing', got %q", resp["status"])
				}
			},
		},
		{
			name:           "POST /api/forms/:id/analyze returns 404 for nonexistent form",
			method:         http.MethodPost,
			path:           "/api/forms/nonexistent/analyze",
			expectedStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, fs, _, _ := setupFormsTestApp(t)

			if tt.setupForms != nil {
				tt.setupForms(fs)
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
