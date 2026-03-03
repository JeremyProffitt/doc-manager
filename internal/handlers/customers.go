package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// CustomersHandler holds dependencies for customer-related HTTP handlers.
type CustomersHandler struct {
	customerStore store.CustomerStore
}

// NewCustomersHandler creates a new CustomersHandler with the given store.
func NewCustomersHandler(cs store.CustomerStore) *CustomersHandler {
	return &CustomersHandler{customerStore: cs}
}

// ListCustomers renders the customers list page.
func (h *CustomersHandler) ListCustomers(c *fiber.Ctx) error {
	userEmail, _ := c.Locals("userEmail").(string)

	customers, err := h.customerStore.ListCustomers()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to list customers"})
	}

	return c.Render("customers/list", fiber.Map{
		"UserEmail":  userEmail,
		"Customers":  customers,
	}, "layouts/base")
}

// GetCustomer renders a single customer detail page.
func (h *CustomersHandler) GetCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	userEmail, _ := c.Locals("userEmail").(string)

	customer, err := h.customerStore.GetCustomer(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get customer"})
	}
	if customer == nil {
		return c.Status(404).JSON(fiber.Map{"error": "customer not found"})
	}

	return c.Render("customers/view", fiber.Map{
		"UserEmail": userEmail,
		"Customer":  customer,
	}, "layouts/base")
}

// NewCustomer renders the add customer form.
func (h *CustomersHandler) NewCustomer(c *fiber.Ctx) error {
	userEmail, _ := c.Locals("userEmail").(string)

	return c.Render("customers/edit", fiber.Map{
		"UserEmail": userEmail,
		"IsNew":     true,
	}, "layouts/base")
}

// CreateCustomer handles the form submission for creating a new customer.
func (h *CustomersHandler) CreateCustomer(c *fiber.Ctx) error {
	customer := &models.Customer{
		ID:       uuid.New().String(),
		Name:     c.FormValue("name"),
		Business: c.FormValue("business"),
		Address:  c.FormValue("address"),
		City:     c.FormValue("city"),
		State:    c.FormValue("state"),
		Zip:      c.FormValue("zip"),
		Phone:    c.FormValue("phone"),
	}

	if err := h.customerStore.CreateCustomer(customer); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create customer"})
	}

	return c.Redirect("/customers")
}

// UpdateCustomer handles updating an existing customer via JSON API.
func (h *CustomersHandler) UpdateCustomer(c *fiber.Ctx) error {
	id := c.Params("id")

	existing, err := h.customerStore.GetCustomer(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get customer"})
	}
	if existing == nil {
		return c.Status(404).JSON(fiber.Map{"error": "customer not found"})
	}

	var updated models.Customer
	if err := c.BodyParser(&updated); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	updated.ID = id

	if err := h.customerStore.UpdateCustomer(&updated); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update customer"})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// DeleteCustomer handles deleting a customer via JSON API.
func (h *CustomersHandler) DeleteCustomer(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.customerStore.DeleteCustomer(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete customer"})
	}

	return c.JSON(fiber.Map{"status": "deleted"})
}

// ListCustomersAPI returns all customers as a JSON array.
func (h *CustomersHandler) ListCustomersAPI(c *fiber.Ctx) error {
	customers, err := h.customerStore.ListCustomers()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to list customers"})
	}

	return c.JSON(customers)
}
