package services

import (
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// --- Mock stores for analysis tests ---

type mockFormStore struct {
	mu    sync.RWMutex
	forms map[string]*models.Form
}

func newMockFormStore() *mockFormStore {
	return &mockFormStore{forms: make(map[string]*models.Form)}
}

func (m *mockFormStore) CreateForm(form *models.Form) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f := *form
	m.forms[form.ID] = &f
	return nil
}

func (m *mockFormStore) GetForm(id string) (*models.Form, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if f, ok := m.forms[id]; ok {
		c := *f
		return &c, nil
	}
	return nil, nil
}

func (m *mockFormStore) ListForms(userId string) ([]models.Form, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []models.Form
	for _, f := range m.forms {
		if f.UserID == userId {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *mockFormStore) UpdateForm(form *models.Form) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f := *form
	m.forms[form.ID] = &f
	return nil
}

func (m *mockFormStore) DeleteForm(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.forms, id)
	return nil
}

var _ store.FormStore = (*mockFormStore)(nil)

type mockFieldStore struct {
	mu         sync.RWMutex
	placements map[string][]models.FieldPlacement
}

func newMockFieldStore() *mockFieldStore {
	return &mockFieldStore{placements: make(map[string][]models.FieldPlacement)}
}

func (m *mockFieldStore) SaveFieldPlacement(placement *models.FieldPlacement) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing := m.placements[placement.FormID]
	maxVersion := 0
	for _, p := range existing {
		if p.Version > maxVersion {
			maxVersion = p.Version
		}
	}
	newVersion := maxVersion + 1
	p := *placement
	p.Version = newVersion
	m.placements[placement.FormID] = append(existing, p)
	return newVersion, nil
}

func (m *mockFieldStore) GetLatestFieldPlacement(formId string) (*models.FieldPlacement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions := m.placements[formId]
	if len(versions) == 0 {
		return nil, nil
	}
	latest := versions[0]
	for _, v := range versions {
		if v.Version > latest.Version {
			latest = v
		}
	}
	c := latest
	return &c, nil
}

func (m *mockFieldStore) GetFieldPlacement(formId string, version int) (*models.FieldPlacement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.placements[formId] {
		if p.Version == version {
			c := p
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockFieldStore) ListVersions(formId string) ([]models.FieldPlacement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions := m.placements[formId]
	result := make([]models.FieldPlacement, len(versions))
	for i, v := range versions {
		result[i] = models.FieldPlacement{
			FormID:    v.FormID,
			Version:   v.Version,
			CreatedAt: v.CreatedAt,
			Source:    v.Source,
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version > result[j].Version
	})
	return result, nil
}

func (m *mockFieldStore) DeleteAllVersions(formId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.placements, formId)
	return nil
}

var _ store.FieldStore = (*mockFieldStore)(nil)

type mockSettingsStore struct {
	mu       sync.RWMutex
	settings map[string]*models.Setting
}

func newMockSettingsStore() *mockSettingsStore {
	return &mockSettingsStore{settings: make(map[string]*models.Setting)}
}

func (m *mockSettingsStore) GetSetting(key string) (*models.Setting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.settings[key]; ok {
		c := *s
		return &c, nil
	}
	return nil, nil
}

func (m *mockSettingsStore) PutSetting(setting *models.Setting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := *setting
	m.settings[setting.Key] = &s
	return nil
}

var _ store.SettingsStore = (*mockSettingsStore)(nil)

// mockS3Svc implements S3ServiceInterface for analysis tests.
type mockS3Svc struct {
	data    []byte
	dataErr error
}

func (m *mockS3Svc) GenerateUploadURL(formId, filename, contentType string) (string, string, error) {
	return "https://example.com/upload", "forms/" + formId + "/" + filename, nil
}

func (m *mockS3Svc) GenerateDownloadURL(s3Key string) (string, error) {
	return "https://example.com/download/" + s3Key, nil
}

func (m *mockS3Svc) GetObject(s3Key string) ([]byte, error) {
	if m.dataErr != nil {
		return nil, m.dataErr
	}
	return m.data, nil
}

func (m *mockS3Svc) DeleteObject(s3Key string) error {
	return nil
}

var _ S3ServiceInterface = (*mockS3Svc)(nil)

func TestAnalysisService_AnalyzeForm(t *testing.T) {
	tests := []struct {
		name            string
		formID          string
		setupForm       func(fs *mockFormStore)
		setupSettings   func(ss *mockSettingsStore)
		s3Data          []byte
		s3Err           error
		bedrockResponse []byte
		bedrockErr      error
		wantErr         bool
		checkForm       func(t *testing.T, fs *mockFormStore)
		checkFields     func(t *testing.T, flds *mockFieldStore)
	}{
		{
			name:   "single-page image analyzes and saves as version 1 with source ai_analysis",
			formID: "form-1",
			setupForm: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-1",
					UserID:      "test@example.com",
					Name:        "test.png",
					Status:      "uploaded",
					S3Key:       "forms/form-1/test.png",
					ContentType: "image/png",
				})
			},
			setupSettings: func(ss *mockSettingsStore) {
				fields, _ := json.Marshal([]models.FieldDefinition{
					{Name: "Name", Type: "text"},
					{Name: "Date", Type: "date"},
				})
				ss.PutSetting(&models.Setting{Key: "standard_fields", Value: string(fields)})
			},
			s3Data: []byte("fake-image-bytes"),
			bedrockResponse: bedrockResponse(`[
				{"field_name":"Name","page":1,"x":10.0,"y":20.0,"width":30.0,"height":5.0,"confidence":0.95,"reasoning":"Found name label"},
				{"field_name":"Date","page":1,"x":60.0,"y":20.0,"width":20.0,"height":5.0,"confidence":0.88,"reasoning":"Found date label"}
			]`),
			wantErr: false,
			checkForm: func(t *testing.T, fs *mockFormStore) {
				t.Helper()
				form, _ := fs.GetForm("form-1")
				if form.Status != "analyzed" {
					t.Errorf("expected form status 'analyzed', got %q", form.Status)
				}
			},
			checkFields: func(t *testing.T, flds *mockFieldStore) {
				t.Helper()
				placement, _ := flds.GetLatestFieldPlacement("form-1")
				if placement == nil {
					t.Fatal("expected field placement to be saved")
				}
				if placement.Version != 1 {
					t.Errorf("expected version 1, got %d", placement.Version)
				}
				if placement.Source != "ai_analysis" {
					t.Errorf("expected source 'ai_analysis', got %q", placement.Source)
				}
				if len(placement.Fields) != 2 {
					t.Errorf("expected 2 fields, got %d", len(placement.Fields))
				}
			},
		},
		{
			name:   "form status transitions uploaded to analyzing to analyzed",
			formID: "form-2",
			setupForm: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-2",
					UserID:      "test@example.com",
					Name:        "test.png",
					Status:      "uploaded",
					S3Key:       "forms/form-2/test.png",
					ContentType: "image/png",
				})
			},
			setupSettings: func(ss *mockSettingsStore) {
				fields, _ := json.Marshal([]models.FieldDefinition{
					{Name: "Name", Type: "text"},
				})
				ss.PutSetting(&models.Setting{Key: "standard_fields", Value: string(fields)})
			},
			s3Data: []byte("fake-image-bytes"),
			bedrockResponse: bedrockResponse(`[
				{"field_name":"Name","page":1,"x":10.0,"y":20.0,"width":30.0,"height":5.0,"confidence":0.95,"reasoning":"ok"}
			]`),
			wantErr: false,
			checkForm: func(t *testing.T, fs *mockFormStore) {
				t.Helper()
				form, _ := fs.GetForm("form-2")
				if form.Status != "analyzed" {
					t.Errorf("expected final status 'analyzed', got %q", form.Status)
				}
			},
		},
		{
			name:   "analysis failure sets form status to error",
			formID: "form-3",
			setupForm: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-3",
					UserID:      "test@example.com",
					Name:        "test.png",
					Status:      "uploaded",
					S3Key:       "forms/form-3/test.png",
					ContentType: "image/png",
				})
			},
			setupSettings: func(ss *mockSettingsStore) {
				fields, _ := json.Marshal([]models.FieldDefinition{
					{Name: "Name", Type: "text"},
				})
				ss.PutSetting(&models.Setting{Key: "standard_fields", Value: string(fields)})
			},
			s3Data:     []byte("fake-image-bytes"),
			bedrockErr: errors.New("bedrock failed"),
			wantErr:    true,
			checkForm: func(t *testing.T, fs *mockFormStore) {
				t.Helper()
				form, _ := fs.GetForm("form-3")
				if form.Status != "error" {
					t.Errorf("expected form status 'error' after failure, got %q", form.Status)
				}
			},
		},
		{
			name:   "AI returns fields and all are saved in FieldPlacement",
			formID: "form-4",
			setupForm: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-4",
					UserID:      "test@example.com",
					Name:        "test.png",
					Status:      "uploaded",
					S3Key:       "forms/form-4/test.png",
					ContentType: "image/png",
				})
			},
			setupSettings: func(ss *mockSettingsStore) {
				fields, _ := json.Marshal([]models.FieldDefinition{
					{Name: "FirstName", Type: "text"},
					{Name: "LastName", Type: "text"},
					{Name: "DOB", Type: "date"},
				})
				ss.PutSetting(&models.Setting{Key: "standard_fields", Value: string(fields)})
			},
			s3Data: []byte("fake-image-bytes"),
			bedrockResponse: bedrockResponse(`[
				{"field_name":"FirstName","page":1,"x":10.0,"y":20.0,"width":30.0,"height":5.0,"confidence":0.95,"reasoning":"ok"},
				{"field_name":"LastName","page":1,"x":10.0,"y":30.0,"width":30.0,"height":5.0,"confidence":0.92,"reasoning":"ok"},
				{"field_name":"DOB","page":1,"x":10.0,"y":40.0,"width":20.0,"height":5.0,"confidence":0.88,"reasoning":"ok"}
			]`),
			wantErr: false,
			checkFields: func(t *testing.T, flds *mockFieldStore) {
				t.Helper()
				placement, _ := flds.GetLatestFieldPlacement("form-4")
				if placement == nil {
					t.Fatal("expected field placement to be saved")
				}
				if len(placement.Fields) != 3 {
					t.Errorf("expected 3 fields saved, got %d", len(placement.Fields))
				}
				fieldNames := make(map[string]bool)
				for _, f := range placement.Fields {
					fieldNames[f.FieldName] = true
				}
				for _, expected := range []string{"FirstName", "LastName", "DOB"} {
					if !fieldNames[expected] {
						t.Errorf("expected field %q to be saved", expected)
					}
				}
			},
		},
		{
			name:   "form not found returns error",
			formID: "nonexistent",
			setupForm: func(fs *mockFormStore) {
				// no form created
			},
			wantErr: true,
		},
		{
			name:   "S3 fetch failure sets form status to error",
			formID: "form-5",
			setupForm: func(fs *mockFormStore) {
				fs.CreateForm(&models.Form{
					ID:          "form-5",
					UserID:      "test@example.com",
					Name:        "test.png",
					Status:      "uploaded",
					S3Key:       "forms/form-5/test.png",
					ContentType: "image/png",
				})
			},
			setupSettings: func(ss *mockSettingsStore) {
				fields, _ := json.Marshal([]models.FieldDefinition{
					{Name: "Name", Type: "text"},
				})
				ss.PutSetting(&models.Setting{Key: "standard_fields", Value: string(fields)})
			},
			s3Err:   errors.New("s3 fetch failed"),
			wantErr: true,
			checkForm: func(t *testing.T, fs *mockFormStore) {
				t.Helper()
				form, _ := fs.GetForm("form-5")
				if form.Status != "error" {
					t.Errorf("expected form status 'error', got %q", form.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formStore := newMockFormStore()
			fieldStore := newMockFieldStore()
			settingsStore := newMockSettingsStore()

			if tt.setupForm != nil {
				tt.setupForm(formStore)
			}
			if tt.setupSettings != nil {
				tt.setupSettings(settingsStore)
			}

			bedrockClient := &mockBedrockClient{
				response: tt.bedrockResponse,
				err:      tt.bedrockErr,
			}
			bedrockSvc := NewBedrockService(bedrockClient, "anthropic.claude-sonnet-4-20250514")

			s3Svc := &mockS3Svc{
				data:    tt.s3Data,
				dataErr: tt.s3Err,
			}

			analysisSvc := NewAnalysisService(bedrockSvc, s3Svc, formStore, fieldStore, settingsStore)

			err := analysisSvc.AnalyzeForm(tt.formID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AnalyzeForm() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.checkForm != nil {
				tt.checkForm(t, formStore)
			}
			if tt.checkFields != nil {
				tt.checkFields(t, fieldStore)
			}
		})
	}
}
