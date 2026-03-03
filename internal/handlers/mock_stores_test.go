package handlers

import (
	"errors"
	"sort"
	"sync"

	"github.com/JeremyProffitt/doc-manager/internal/models"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

// --- Mock FormStore for handler tests ---

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
	if _, exists := m.forms[form.ID]; exists {
		return errors.New("form already exists")
	}
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
	result := make([]models.Form, 0)
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

// --- Mock FieldStore for handler tests ---

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

// --- Mock CustomerStore for handler tests ---

type mockCustomerStore struct {
	mu        sync.RWMutex
	customers map[string]*models.Customer
}

func newMockCustomerStore() *mockCustomerStore {
	return &mockCustomerStore{customers: make(map[string]*models.Customer)}
}

func (m *mockCustomerStore) CreateCustomer(customer *models.Customer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.customers[customer.ID]; exists {
		return errors.New("customer already exists")
	}
	c := *customer
	m.customers[customer.ID] = &c
	return nil
}

func (m *mockCustomerStore) GetCustomer(id string) (*models.Customer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if c, ok := m.customers[id]; ok {
		copy := *c
		return &copy, nil
	}
	return nil, nil
}

func (m *mockCustomerStore) ListCustomers() ([]models.Customer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]models.Customer, 0, len(m.customers))
	for _, c := range m.customers {
		result = append(result, *c)
	}
	return result, nil
}

func (m *mockCustomerStore) UpdateCustomer(customer *models.Customer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := *customer
	m.customers[customer.ID] = &c
	return nil
}

func (m *mockCustomerStore) DeleteCustomer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.customers, id)
	return nil
}

var _ store.CustomerStore = (*mockCustomerStore)(nil)

// --- Mock SettingsStore for handler tests ---

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

// --- Mock S3 Service for handler tests ---

type mockS3Service struct{}

func newMockS3Service() *mockS3Service {
	return &mockS3Service{}
}

func (m *mockS3Service) GenerateUploadURL(formId, filename, contentType string) (string, string, error) {
	s3Key := "forms/" + formId + "/" + filename
	return "https://s3.amazonaws.com/test-bucket/" + s3Key + "?signed=true", s3Key, nil
}

func (m *mockS3Service) GenerateDownloadURL(s3Key string) (string, error) {
	return "https://s3.amazonaws.com/test-bucket/" + s3Key + "?signed=true", nil
}

func (m *mockS3Service) GetObject(s3Key string) ([]byte, error) {
	return []byte("test content"), nil
}

func (m *mockS3Service) DeleteObject(s3Key string) error {
	return nil
}

// --- Mock Analysis Service for handler tests ---

type mockAnalysisService struct {
	err error
}

func newMockAnalysisService() *mockAnalysisService {
	return &mockAnalysisService{}
}

func (m *mockAnalysisService) AnalyzeForm(formID string) error {
	return m.err
}

var _ AnalysisServiceInterface = (*mockAnalysisService)(nil)
