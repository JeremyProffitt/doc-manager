package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/JeremyProffitt/doc-manager/internal/services"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// PopulateHandler holds dependencies for form population and PDF download handlers.
type PopulateHandler struct {
	formStore     store.FormStore
	fieldStore    store.FieldStore
	customerStore store.CustomerStore
	s3Service     S3ServiceInterface
	pdfService    *services.PDFService
}

// NewPopulateHandler creates a new PopulateHandler with the given dependencies.
func NewPopulateHandler(
	fs store.FormStore,
	flds store.FieldStore,
	cs store.CustomerStore,
	s3 S3ServiceInterface,
	pdf *services.PDFService,
) *PopulateHandler {
	return &PopulateHandler{
		formStore:     fs,
		fieldStore:    flds,
		customerStore: cs,
		s3Service:     s3,
		pdfService:    pdf,
	}
}

// PreviewPopulatedForm renders an HTML preview of the form with customer data overlaid.
func (h *PopulateHandler) PreviewPopulatedForm(c *fiber.Ctx) error {
	formID := c.Params("id")
	custID := c.Params("custId")
	userEmail, _ := c.Locals("userEmail").(string)

	form, err := h.formStore.GetForm(formID)
	if err != nil {
		return c.Status(500).SendString("Failed to get form")
	}
	if form == nil {
		return c.Status(404).SendString("Form not found")
	}

	customer, err := h.customerStore.GetCustomer(custID)
	if err != nil {
		return c.Status(500).SendString("Failed to get customer")
	}
	if customer == nil {
		return c.Status(404).SendString("Customer not found")
	}

	placement, _ := h.fieldStore.GetLatestFieldPlacement(formID)

	// Generate S3 download URL for the form image.
	var imageURL string
	if form.S3Key != "" {
		imageURL, _ = h.s3Service.GenerateDownloadURL(form.S3Key)
	}

	// Get all customers for the selector dropdown.
	customers, _ := h.customerStore.ListCustomers()

	// Serialize placement data to JSON for JavaScript overlay rendering.
	placementJSON := "null"
	if placement != nil {
		data, err := json.Marshal(placement)
		if err == nil {
			placementJSON = string(data)
		}
	}

	// Serialize customer data to JSON for JavaScript overlay rendering.
	customerJSON := "null"
	if customer != nil {
		data, err := json.Marshal(customer)
		if err == nil {
			customerJSON = string(data)
		}
	}

	return c.Render("populate/preview", fiber.Map{
		"Form":          form,
		"Customer":      customer,
		"Customers":     customers,
		"Placement":     placement,
		"PlacementJSON": placementJSON,
		"CustomerJSON":  customerJSON,
		"ImageURL":      imageURL,
		"UserEmail":     userEmail,
	})
}

// DownloadPopulatedForm generates a PDF with customer data overlaid and returns it as a download.
func (h *PopulateHandler) DownloadPopulatedForm(c *fiber.Ctx) error {
	formID := c.Params("id")
	custID := c.Params("custId")

	form, err := h.formStore.GetForm(formID)
	if err != nil {
		return c.Status(500).SendString("Failed to get form")
	}
	if form == nil {
		return c.Status(404).SendString("Form not found")
	}

	customer, err := h.customerStore.GetCustomer(custID)
	if err != nil {
		return c.Status(500).SendString("Failed to get customer")
	}
	if customer == nil {
		return c.Status(404).SendString("Customer not found")
	}

	placement, _ := h.fieldStore.GetLatestFieldPlacement(formID)

	// Fetch form image from S3.
	imageData, err := h.s3Service.GetObject(form.S3Key)
	if err != nil {
		return c.Status(500).SendString("Failed to fetch form")
	}

	// Generate populated PDF.
	pdfBytes, err := h.pdfService.GeneratePopulatedPDF(imageData, form.ContentType, placement, customer)
	if err != nil {
		return c.Status(500).SendString("Failed to generate PDF")
	}

	// Set headers for PDF download.
	filename := fmt.Sprintf("%s_%s.pdf", form.Name, customer.Name)
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	return c.Send(pdfBytes)
}
