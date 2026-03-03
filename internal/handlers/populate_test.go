package handlers

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/services"
)

// mockS3ServiceWithImage returns valid PNG bytes from GetObject for PDF generation.
type mockS3ServiceWithImage struct {
	imageData []byte
}

func newMockS3ServiceWithImage() *mockS3ServiceWithImage {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return &mockS3ServiceWithImage{imageData: buf.Bytes()}
}

func (m *mockS3ServiceWithImage) GenerateUploadURL(formId, filename, contentType string) (string, string, error) {
	s3Key := "forms/" + formId + "/" + filename
	return "https://s3.amazonaws.com/test-bucket/" + s3Key + "?signed=true", s3Key, nil
}

func (m *mockS3ServiceWithImage) GenerateDownloadURL(s3Key string) (string, error) {
	return "https://s3.amazonaws.com/test-bucket/" + s3Key + "?signed=true", nil
}

func (m *mockS3ServiceWithImage) GetObject(s3Key string) ([]byte, error) {
	return m.imageData, nil
}

func (m *mockS3ServiceWithImage) DeleteObject(s3Key string) error {
	return nil
}

func setupPopulateTestApp(t *testing.T) (*fiber.App, *mockFormStore, *mockFieldStore, *mockCustomerStore) {
	t.Helper()
	engine := html.New("../../templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	fs := newMockFormStore()
	flds := newMockFieldStore()
	cs := newMockCustomerStore()
	s3svc := newMockS3ServiceWithImage()
	pdfSvc := services.NewPDFService()

	h := NewPopulateHandler(fs, flds, cs, s3svc, pdfSvc)

	// Simulate auth middleware setting userEmail.
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userEmail", "test@example.com")
		return c.Next()
	})

	app.Get("/forms/:id/populate/:custId", h.PreviewPopulatedForm)
	app.Get("/forms/:id/download/:custId", h.DownloadPopulatedForm)

	return app, fs, flds, cs
}

func TestPopulateHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		setupData      func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
		checkHeaders   func(t *testing.T, resp *http.Response)
	}{
		{
			name:   "GET /forms/:id/populate/:custId returns 200 with HTML preview",
			method: http.MethodGet,
			path:   "/forms/form-1/populate/cust-1",
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-1",
					UserID:      "test@example.com",
					Name:        "W-9 Form",
					Status:      "analyzed",
					S3Key:       "forms/form-1/w9.png",
					ContentType: "image/png",
					FontFamily:  "Helvetica",
					FontSize:    12,
				})
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:     "form-1",
					Source:     "ai_analysis",
					FontFamily: "Helvetica",
					FontSize:   12,
					Fields: []models.Field{
						{FieldName: "Name", X: 15, Y: 13, Width: 50, Height: 3},
						{FieldName: "Address", X: 15, Y: 25, Width: 50, Height: 3},
					},
				})
				cs.CreateCustomer(&models.Customer{
					ID:      "cust-1",
					Name:    "John Doe",
					Address: "123 Main St",
					City:    "Springfield",
					State:   "IL",
					Zip:     "62701",
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "W-9 Form") {
					t.Error("expected preview page to contain form name 'W-9 Form'")
				}
				if !strings.Contains(body, "John Doe") {
					t.Error("expected preview page to contain customer name 'John Doe'")
				}
				if !strings.Contains(body, "form-1") {
					t.Error("expected preview page to contain form ID 'form-1'")
				}
				if !strings.Contains(body, "cust-1") {
					t.Error("expected preview page to contain customer ID 'cust-1'")
				}
			},
		},
		{
			name:   "GET /forms/:id/download/:custId returns 200 with PDF",
			method: http.MethodGet,
			path:   "/forms/form-1/download/cust-1",
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-1",
					UserID:      "test@example.com",
					Name:        "Invoice",
					Status:      "analyzed",
					S3Key:       "forms/form-1/invoice.png",
					ContentType: "image/png",
					FontFamily:  "Helvetica",
					FontSize:    12,
				})
				flds.SaveFieldPlacement(&models.FieldPlacement{
					FormID:     "form-1",
					Source:     "ai_analysis",
					FontFamily: "Helvetica",
					FontSize:   12,
					Fields: []models.Field{
						{FieldName: "Name", X: 10, Y: 20, Width: 30, Height: 3},
					},
				})
				cs.CreateCustomer(&models.Customer{
					ID:   "cust-1",
					Name: "Jane Smith",
				})
			},
			expectedStatus: 200,
			checkHeaders: func(t *testing.T, resp *http.Response) {
				t.Helper()
				ct := resp.Header.Get("Content-Type")
				if ct != "application/pdf" {
					t.Errorf("expected Content-Type application/pdf, got %q", ct)
				}
				cd := resp.Header.Get("Content-Disposition")
				if !strings.Contains(cd, "attachment") {
					t.Errorf("expected Content-Disposition to contain 'attachment', got %q", cd)
				}
				if !strings.Contains(cd, ".pdf") {
					t.Errorf("expected Content-Disposition to contain '.pdf', got %q", cd)
				}
			},
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.HasPrefix(body, "%PDF") {
					t.Error("expected response body to start with %PDF")
				}
			},
		},
		{
			name:           "Populate with non-existent form returns 404",
			method:         http.MethodGet,
			path:           "/forms/no-such-form/populate/cust-1",
			expectedStatus: 404,
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				cs.CreateCustomer(&models.Customer{ID: "cust-1", Name: "Test"})
			},
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Form not found") {
					t.Errorf("expected 'Form not found' in body, got %q", body)
				}
			},
		},
		{
			name:   "Populate with non-existent customer returns 404",
			method: http.MethodGet,
			path:   "/forms/form-1/populate/no-such-cust",
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-1",
					UserID:      "test@example.com",
					Name:        "Test Form",
					Status:      "uploaded",
					S3Key:       "forms/form-1/test.png",
					ContentType: "image/png",
					FontFamily:  "Helvetica",
					FontSize:    12,
				})
			},
			expectedStatus: 404,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Customer not found") {
					t.Errorf("expected 'Customer not found' in body, got %q", body)
				}
			},
		},
		{
			name:   "Download with non-existent form returns 404",
			method: http.MethodGet,
			path:   "/forms/no-form/download/cust-1",
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				cs.CreateCustomer(&models.Customer{ID: "cust-1", Name: "Test"})
			},
			expectedStatus: 404,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Form not found") {
					t.Error("expected 'Form not found' in body")
				}
			},
		},
		{
			name:   "Download with non-existent customer returns 404",
			method: http.MethodGet,
			path:   "/forms/form-1/download/no-cust",
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-1",
					UserID:      "test@example.com",
					Name:        "Test Form",
					Status:      "uploaded",
					S3Key:       "forms/form-1/test.png",
					ContentType: "image/png",
					FontFamily:  "Helvetica",
					FontSize:    12,
				})
			},
			expectedStatus: 404,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Customer not found") {
					t.Error("expected 'Customer not found' in body")
				}
			},
		},
		{
			name:   "Populate form with no field placements still returns 200",
			method: http.MethodGet,
			path:   "/forms/form-2/populate/cust-1",
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-2",
					UserID:      "test@example.com",
					Name:        "Empty Form",
					Status:      "uploaded",
					S3Key:       "forms/form-2/empty.png",
					ContentType: "image/png",
					FontFamily:  "Helvetica",
					FontSize:    12,
				})
				cs.CreateCustomer(&models.Customer{
					ID:   "cust-1",
					Name: "Test Customer",
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Empty Form") {
					t.Error("expected page to contain form name 'Empty Form'")
				}
				if !strings.Contains(body, "Test Customer") {
					t.Error("expected page to contain customer name 'Test Customer'")
				}
			},
		},
		{
			name:   "Download form with no field placements returns valid PDF (original form only)",
			method: http.MethodGet,
			path:   "/forms/form-3/download/cust-1",
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-3",
					UserID:      "test@example.com",
					Name:        "No Fields Form",
					Status:      "uploaded",
					S3Key:       "forms/form-3/plain.png",
					ContentType: "image/png",
					FontFamily:  "Helvetica",
					FontSize:    12,
				})
				cs.CreateCustomer(&models.Customer{
					ID:   "cust-1",
					Name: "Test Customer",
				})
			},
			expectedStatus: 200,
			checkHeaders: func(t *testing.T, resp *http.Response) {
				t.Helper()
				ct := resp.Header.Get("Content-Type")
				if ct != "application/pdf" {
					t.Errorf("expected Content-Type application/pdf, got %q", ct)
				}
			},
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.HasPrefix(body, "%PDF") {
					t.Error("expected PDF output even with no field placements")
				}
			},
		},
		{
			name:   "Preview page includes customer selector with all customers",
			method: http.MethodGet,
			path:   "/forms/form-1/populate/cust-1",
			setupData: func(fs *mockFormStore, flds *mockFieldStore, cs *mockCustomerStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-1",
					UserID:      "test@example.com",
					Name:        "Selector Test",
					Status:      "uploaded",
					S3Key:       "forms/form-1/test.png",
					ContentType: "image/png",
					FontFamily:  "Helvetica",
					FontSize:    12,
				})
				cs.CreateCustomer(&models.Customer{ID: "cust-1", Name: "Customer A"})
				cs.CreateCustomer(&models.Customer{ID: "cust-2", Name: "Customer B"})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Customer A") {
					t.Error("expected preview to contain 'Customer A'")
				}
				if !strings.Contains(body, "Customer B") {
					t.Error("expected preview to contain 'Customer B'")
				}
				// Should have a download link/button
				if !strings.Contains(body, "download") && !strings.Contains(body, "Download") {
					t.Error("expected preview page to contain a download link or button")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, fs, flds, cs := setupPopulateTestApp(t)

			if tt.setupData != nil {
				tt.setupData(fs, flds, cs)
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, resp.StatusCode, string(body))
				return
			}

			if tt.checkHeaders != nil {
				tt.checkHeaders(t, resp)
			}

			if tt.checkBody != nil {
				body, _ := io.ReadAll(resp.Body)
				tt.checkBody(t, string(body))
			}
		})
	}
}
