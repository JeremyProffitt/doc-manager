package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// EditorHandler holds dependencies for the form field editor page.
type EditorHandler struct {
	formStore  store.FormStore
	fieldStore store.FieldStore
	s3Service  S3ServiceInterface
}

// NewEditorHandler creates a new EditorHandler with the given dependencies.
func NewEditorHandler(fs store.FormStore, flds store.FieldStore, s3 S3ServiceInterface) *EditorHandler {
	return &EditorHandler{
		formStore:  fs,
		fieldStore: flds,
		s3Service:  s3,
	}
}

// EditForm renders the interactive form field editor page.
func (h *EditorHandler) EditForm(c *fiber.Ctx) error {
	formID := c.Params("id")
	userEmail, _ := c.Locals("userEmail").(string)

	// Get form metadata
	form, err := h.formStore.GetForm(formID)
	if err != nil {
		return c.Status(500).SendString("Failed to get form")
	}
	if form == nil {
		return c.Status(404).SendString("Form not found")
	}

	// Get latest field placements (may be nil for new forms)
	placement, _ := h.fieldStore.GetLatestFieldPlacement(formID)

	// Get all versions for history panel
	versions, _ := h.fieldStore.ListVersions(formID)

	// Generate S3 download URL for the form image
	var imageURL string
	if form.S3Key != "" {
		imageURL, _ = h.s3Service.GenerateDownloadURL(form.S3Key)
	}

	// Serialize field placement data to JSON for JavaScript
	fieldsJSON := "null"
	if placement != nil {
		data, err := json.Marshal(placement)
		if err == nil {
			fieldsJSON = string(data)
		}
	}

	versionsJSON := "null"
	if versions != nil && len(versions) > 0 {
		data, err := json.Marshal(versions)
		if err == nil {
			versionsJSON = string(data)
		}
	}

	return c.Render("forms/editor", fiber.Map{
		"Form":         form,
		"Placement":    placement,
		"FieldsJSON":   fieldsJSON,
		"VersionsJSON": versionsJSON,
		"ImageURL":     imageURL,
		"UserEmail":    userEmail,
	})
}
