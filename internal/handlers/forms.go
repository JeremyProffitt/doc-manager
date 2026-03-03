package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// S3ServiceInterface defines the S3 operations needed by FormsHandler.
type S3ServiceInterface interface {
	GenerateUploadURL(formId, filename, contentType string) (string, string, error)
	GenerateDownloadURL(s3Key string) (string, error)
	GetObject(s3Key string) ([]byte, error)
	DeleteObject(s3Key string) error
}

// FormsHandler holds dependencies for form-related HTTP handlers.
type FormsHandler struct {
	formStore  store.FormStore
	fieldStore store.FieldStore
	s3Service  S3ServiceInterface
}

// NewFormsHandler creates a new FormsHandler with the given dependencies.
func NewFormsHandler(fs store.FormStore, flds store.FieldStore, s3svc S3ServiceInterface) *FormsHandler {
	return &FormsHandler{
		formStore:  fs,
		fieldStore: flds,
		s3Service:  s3svc,
	}
}

// ListForms renders the forms list page with all forms for the current user.
func (h *FormsHandler) ListForms(c *fiber.Ctx) error {
	userEmail, _ := c.Locals("userEmail").(string)

	forms, err := h.formStore.ListForms(userEmail)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to list forms"})
	}

	return c.Render("forms/list", fiber.Map{
		"UserEmail": userEmail,
		"Forms":     forms,
	}, "layouts/base")
}

// GetForm renders a single form view page.
func (h *FormsHandler) GetForm(c *fiber.Ctx) error {
	id := c.Params("id")
	userEmail, _ := c.Locals("userEmail").(string)

	form, err := h.formStore.GetForm(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get form"})
	}
	if form == nil {
		return c.Status(404).JSON(fiber.Map{"error": "form not found"})
	}

	return c.Render("forms/view", fiber.Map{
		"UserEmail": userEmail,
		"Form":      form,
	}, "layouts/base")
}

// uploadURLRequest is the request body for GetUploadURL.
type uploadURLRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
}

// GetUploadURL generates a pre-signed S3 upload URL and creates a form record.
func (h *FormsHandler) GetUploadURL(c *fiber.Ctx) error {
	userEmail, _ := c.Locals("userEmail").(string)

	var req uploadURLRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Filename == "" {
		return c.Status(400).JSON(fiber.Map{"error": "filename is required"})
	}
	if req.ContentType == "" {
		req.ContentType = "application/pdf"
	}

	formId := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	// Create form record with uploading status
	form := &models.Form{
		ID:          formId,
		UserID:      userEmail,
		Name:        req.Filename,
		Status:      "uploading",
		ContentType: req.ContentType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.formStore.CreateForm(form); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create form record"})
	}

	// Generate pre-signed upload URL
	uploadURL, s3Key, err := h.s3Service.GenerateUploadURL(formId, req.Filename, req.ContentType)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate upload URL"})
	}

	return c.JSON(fiber.Map{
		"formId":    formId,
		"uploadUrl": uploadURL,
		"s3Key":     s3Key,
	})
}

// uploadCompleteRequest is the request body for UploadComplete.
type uploadCompleteRequest struct {
	S3Key string `json:"s3Key"`
}

// UploadComplete marks a form as uploaded after the client finishes uploading.
func (h *FormsHandler) UploadComplete(c *fiber.Ctx) error {
	id := c.Params("id")

	var req uploadCompleteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	form, err := h.formStore.GetForm(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get form"})
	}
	if form == nil {
		return c.Status(404).JSON(fiber.Map{"error": "form not found"})
	}

	form.Status = "uploaded"
	form.S3Key = req.S3Key
	form.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := h.formStore.UpdateForm(form); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update form"})
	}

	return c.JSON(fiber.Map{
		"status": "uploaded",
		"formId": form.ID,
	})
}

// DeleteForm deletes a form and its associated S3 object and field placements.
func (h *FormsHandler) DeleteForm(c *fiber.Ctx) error {
	id := c.Params("id")

	form, err := h.formStore.GetForm(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get form"})
	}
	if form == nil {
		return c.Status(404).JSON(fiber.Map{"error": "form not found"})
	}

	// Delete S3 object if it exists
	if form.S3Key != "" {
		_ = h.s3Service.DeleteObject(form.S3Key)
	}

	// Delete all field placement versions
	_ = h.fieldStore.DeleteAllVersions(id)

	// Delete the form record
	if err := h.formStore.DeleteForm(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete form"})
	}

	return c.JSON(fiber.Map{"status": "deleted"})
}
