package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// VersionsHandler holds dependencies for field placement version HTTP handlers.
type VersionsHandler struct {
	fieldStore store.FieldStore
}

// NewVersionsHandler creates a new VersionsHandler with the given field store.
func NewVersionsHandler(fs store.FieldStore) *VersionsHandler {
	return &VersionsHandler{fieldStore: fs}
}

// GetCurrentFields returns the latest field placement for a form.
// GET /api/forms/:id/fields
func (h *VersionsHandler) GetCurrentFields(c *fiber.Ctx) error {
	formID := c.Params("id")

	placement, err := h.fieldStore.GetLatestFieldPlacement(formID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get field placement"})
	}
	if placement == nil {
		return c.Status(404).JSON(fiber.Map{"error": "no field placements found"})
	}

	return c.JSON(placement)
}

// ListVersions returns metadata for all versions of field placements for a form.
// GET /api/forms/:id/fields/versions
func (h *VersionsHandler) ListVersions(c *fiber.Ctx) error {
	formID := c.Params("id")

	versions, err := h.fieldStore.ListVersions(formID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to list versions"})
	}

	if versions == nil {
		versions = []models.FieldPlacement{}
	}

	return c.JSON(versions)
}

// GetVersion returns a specific version of field placements for a form.
// GET /api/forms/:id/fields/:version
func (h *VersionsHandler) GetVersion(c *fiber.Ctx) error {
	formID := c.Params("id")
	versionStr := c.Params("version")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid version number"})
	}

	placement, err := h.fieldStore.GetFieldPlacement(formID, version)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get field placement"})
	}
	if placement == nil {
		return c.Status(404).JSON(fiber.Map{"error": "version not found"})
	}

	return c.JSON(placement)
}

// RevertToVersion creates a new version by copying fields and font settings from
// an old version. The source is set to "revert_from_vN".
// POST /api/forms/:id/fields/revert/:v
func (h *VersionsHandler) RevertToVersion(c *fiber.Ctx) error {
	formID := c.Params("id")
	versionStr := c.Params("v")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid version number"})
	}

	// Get the old version to revert to
	oldPlacement, err := h.fieldStore.GetFieldPlacement(formID, version)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get field placement"})
	}
	if oldPlacement == nil {
		return c.Status(404).JSON(fiber.Map{"error": "version not found"})
	}

	// Create a new version with the old version's fields and font settings
	newPlacement := &models.FieldPlacement{
		FormID:     formID,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Source:     fmt.Sprintf("revert_from_v%d", version),
		FontFamily: oldPlacement.FontFamily,
		FontSize:   oldPlacement.FontSize,
		Fields:     oldPlacement.Fields,
	}

	newVersion, err := h.fieldStore.SaveFieldPlacement(newPlacement)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save reverted placement"})
	}

	newPlacement.Version = newVersion
	return c.JSON(newPlacement)
}

// saveFieldsRequest is the request body for SaveFields.
type saveFieldsRequest struct {
	FontFamily string         `json:"fontFamily"`
	FontSize   int            `json:"fontSize"`
	Fields     []models.Field `json:"fields"`
}

// SaveFields saves a new version of field placements from manual edits.
// PUT /api/forms/:id/fields
func (h *VersionsHandler) SaveFields(c *fiber.Ctx) error {
	formID := c.Params("id")

	var req saveFieldsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	placement := &models.FieldPlacement{
		FormID:     formID,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Source:     "manual_edit",
		FontFamily: req.FontFamily,
		FontSize:   req.FontSize,
		Fields:     req.Fields,
	}

	newVersion, err := h.fieldStore.SaveFieldPlacement(placement)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save field placement"})
	}

	placement.Version = newVersion
	return c.JSON(placement)
}
