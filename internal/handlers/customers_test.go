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

func setupCustomersTestApp(t *testing.T) (*fiber.App, *mockCustomerStore) {
	t.Helper()
	engine := html.New("../../templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	cs := newMockCustomerStore()
	h := NewCustomersHandler(cs)

	// Simulate auth middleware setting userEmail
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userEmail", "test@example.com")
		return c.Next()
	})

	app.Get("/customers", h.ListCustomers)
	app.Get("/customers/new", h.NewCustomer)
	app.Get("/customers/:id", h.GetCustomer)
	app.Post("/customers", h.CreateCustomer)
	app.Put("/api/customers/:id", h.UpdateCustomer)
	app.Delete("/api/customers/:id", h.DeleteCustomer)
	app.Get("/api/customers", h.ListCustomersAPI)

	return app, cs
}

func TestCustomersHandlers(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		setupCustomers func(cs *mockCustomerStore)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
		checkRedirect  string
	}{
		{
			name:   "GET /customers returns 200 with HTML",
			method: http.MethodGet,
			path:   "/customers",
			setupCustomers: func(cs *mockCustomerStore) {
				cs.CreateCustomer(&models.Customer{
					ID:   "c1",
					Name: "John Smith",
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "John Smith") {
					t.Error("expected customer list to contain 'John Smith'")
				}
			},
		},
		{
			name:           "GET /customers with no customers returns 200",
			method:         http.MethodGet,
			path:           "/customers",
			expectedStatus: 200,
		},
		{
			name:        "POST /customers creates customer and redirects",
			method:      http.MethodPost,
			path:        "/customers",
			body:        "name=Jane+Doe&business=Doe+Inc&address=123+Main&city=Austin&state=TX&zip=78701&phone=555-1234",
			contentType: "application/x-www-form-urlencoded",
			expectedStatus: 302,
			checkRedirect:  "/customers",
		},
		{
			name:   "PUT /api/customers/:id updates customer",
			method: http.MethodPut,
			path:   "/api/customers/c1",
			body:   `{"name":"Updated Name","business":"Updated Biz","address":"456 Oak","city":"Dallas","state":"TX","zip":"75201","phone":"555-9999"}`,
			contentType: "application/json",
			setupCustomers: func(cs *mockCustomerStore) {
				cs.CreateCustomer(&models.Customer{
					ID:   "c1",
					Name: "Original Name",
				})
			},
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
		{
			name:   "DELETE /api/customers/:id returns 200",
			method: http.MethodDelete,
			path:   "/api/customers/c1",
			setupCustomers: func(cs *mockCustomerStore) {
				cs.CreateCustomer(&models.Customer{
					ID:   "c1",
					Name: "To Delete",
				})
			},
			expectedStatus: 200,
		},
		{
			name:   "GET /api/customers returns JSON array",
			method: http.MethodGet,
			path:   "/api/customers",
			setupCustomers: func(cs *mockCustomerStore) {
				cs.CreateCustomer(&models.Customer{ID: "c1", Name: "Alice"})
				cs.CreateCustomer(&models.Customer{ID: "c2", Name: "Bob"})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				var customers []models.Customer
				if err := json.Unmarshal([]byte(body), &customers); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}
				if len(customers) != 2 {
					t.Errorf("expected 2 customers, got %d", len(customers))
				}
			},
		},
		{
			name:   "GET /customers/:id returns 200 with customer detail",
			method: http.MethodGet,
			path:   "/customers/c1",
			setupCustomers: func(cs *mockCustomerStore) {
				cs.CreateCustomer(&models.Customer{
					ID:       "c1",
					Name:     "John Smith",
					Business: "Smith LLC",
				})
			},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "John Smith") {
					t.Error("expected customer view to contain 'John Smith'")
				}
			},
		},
		{
			name:           "GET /customers/:id non-existent returns 404",
			method:         http.MethodGet,
			path:           "/customers/no-such-id",
			expectedStatus: 404,
		},
		{
			name:           "GET /customers/new returns 200 with add form",
			method:         http.MethodGet,
			path:           "/customers/new",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "Add Customer") {
					t.Error("expected new customer page to contain 'Add Customer'")
				}
			},
		},
		{
			name:        "PUT /api/customers/:id non-existent returns 404",
			method:      http.MethodPut,
			path:        "/api/customers/no-such-id",
			body:        `{"name":"Test"}`,
			contentType: "application/json",
			expectedStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, cs := setupCustomersTestApp(t)

			if tt.setupCustomers != nil {
				tt.setupCustomers(cs)
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
		})
	}
}
