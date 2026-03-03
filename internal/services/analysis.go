package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// S3ServiceInterface defines the S3 operations needed by AnalysisService.
type S3ServiceInterface interface {
	GenerateUploadURL(formId, filename, contentType string) (string, string, error)
	GenerateDownloadURL(s3Key string) (string, error)
	GetObject(s3Key string) ([]byte, error)
	DeleteObject(s3Key string) error
}

// AnalysisService orchestrates form analysis using Bedrock AI.
type AnalysisService struct {
	bedrockService *BedrockService
	s3Service      S3ServiceInterface
	formStore      store.FormStore
	fieldStore     store.FieldStore
	settingsStore  store.SettingsStore
}

// NewAnalysisService creates a new AnalysisService with the given dependencies.
func NewAnalysisService(bs *BedrockService, s3 S3ServiceInterface, fs store.FormStore, flds store.FieldStore, ss store.SettingsStore) *AnalysisService {
	return &AnalysisService{
		bedrockService: bs,
		s3Service:      s3,
		formStore:      fs,
		fieldStore:     flds,
		settingsStore:  ss,
	}
}

// AnalyzeForm orchestrates the full analysis flow:
// 1. Fetch the form and update status to "analyzing"
// 2. Fetch the form image from S3
// 3. Get standard fields from settings
// 4. Send to Bedrock for analysis
// 5. Save results as a new version in FieldPlacements
// 6. Update form status to "analyzed"
//
// On failure, the form status is set to "error" and the error is returned.
func (a *AnalysisService) AnalyzeForm(formID string) error {
	// Fetch the form
	form, err := a.formStore.GetForm(formID)
	if err != nil {
		return fmt.Errorf("getting form: %w", err)
	}
	if form == nil {
		return fmt.Errorf("form not found: %s", formID)
	}

	// Update status to analyzing
	form.Status = "analyzing"
	form.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := a.formStore.UpdateForm(form); err != nil {
		return fmt.Errorf("updating form status to analyzing: %w", err)
	}

	// Run the analysis; on any error, set status to "error"
	if err := a.runAnalysis(form); err != nil {
		form.Status = "error"
		form.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		_ = a.formStore.UpdateForm(form)
		return err
	}

	// Update status to analyzed
	form.Status = "analyzed"
	form.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := a.formStore.UpdateForm(form); err != nil {
		return fmt.Errorf("updating form status to analyzed: %w", err)
	}

	return nil
}

// runAnalysis performs the actual S3 fetch, Bedrock analysis, and field saving.
func (a *AnalysisService) runAnalysis(form *models.Form) error {
	// Fetch image from S3
	imageData, err := a.s3Service.GetObject(form.S3Key)
	if err != nil {
		return fmt.Errorf("fetching form image from S3: %w", err)
	}

	// Get standard fields from settings
	standardFields, err := a.getStandardFieldNames()
	if err != nil {
		return fmt.Errorf("getting standard fields: %w", err)
	}

	// Send to Bedrock for analysis
	fields, err := a.bedrockService.AnalyzeForm(imageData, form.ContentType, standardFields)
	if err != nil {
		return fmt.Errorf("analyzing form with bedrock: %w", err)
	}

	// Save results as a new field placement version
	placement := &models.FieldPlacement{
		FormID:    form.ID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    "ai_analysis",
		Fields:    fields,
	}

	if _, err := a.fieldStore.SaveFieldPlacement(placement); err != nil {
		return fmt.Errorf("saving field placement: %w", err)
	}

	return nil
}

// getStandardFieldNames retrieves the list of standard field names from settings.
func (a *AnalysisService) getStandardFieldNames() ([]string, error) {
	setting, err := a.settingsStore.GetSetting("standard_fields")
	if err != nil {
		return nil, fmt.Errorf("getting standard_fields setting: %w", err)
	}
	if setting == nil {
		return []string{}, nil
	}

	var defs []models.FieldDefinition
	if err := json.Unmarshal([]byte(setting.Value), &defs); err != nil {
		return nil, fmt.Errorf("parsing standard_fields: %w", err)
	}

	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	return names, nil
}
