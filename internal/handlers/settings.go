package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// SettingsHandler holds dependencies for settings-related HTTP handlers.
type SettingsHandler struct {
	settingsStore store.SettingsStore
}

// NewSettingsHandler creates a new SettingsHandler with the given store.
func NewSettingsHandler(ss store.SettingsStore) *SettingsHandler {
	return &SettingsHandler{settingsStore: ss}
}

// GetFields renders the field configuration settings page.
func (h *SettingsHandler) GetFields(c *fiber.Ctx) error {
	userEmail, _ := c.Locals("userEmail").(string)

	setting, err := h.settingsStore.GetSetting("standard_fields")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get settings"})
	}

	var fields []models.FieldDefinition
	if setting != nil {
		if err := json.Unmarshal([]byte(setting.Value), &fields); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to parse settings"})
		}
	}

	return c.Render("settings/fields", fiber.Map{
		"UserEmail": userEmail,
		"Fields":    fields,
	}, "layouts/base")
}

// UpdateFields handles updating the standard fields configuration via JSON API.
func (h *SettingsHandler) UpdateFields(c *fiber.Ctx) error {
	var fields []models.FieldDefinition
	if err := c.BodyParser(&fields); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	value, err := json.Marshal(fields)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to serialize fields"})
	}

	// Delete existing setting and create a new one (PutSetting may have condition)
	setting := &models.Setting{
		Key:   "standard_fields",
		Value: string(value),
	}

	if err := h.settingsStore.PutSetting(setting); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update settings"})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}
