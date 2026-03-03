package store

import (
	"encoding/json"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

func TestNewSeeder(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()

	seeder := NewSeeder(us, cs, ss)
	if seeder == nil {
		t.Fatal("NewSeeder returned nil")
	}
}

func TestSeedUser_CreatesUserWithBcryptHash(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	email := "test@example.com"
	password := "TestPassword123!"
	name := "Test User"

	err := seeder.SeedUser(email, password, name)
	if err != nil {
		t.Fatalf("SeedUser returned error: %v", err)
	}

	// Verify user was created in the store
	user, err := us.GetUser(email)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if user == nil {
		t.Fatal("User was not created in the store")
	}

	// Verify fields
	if user.Email != email {
		t.Errorf("Email = %q, want %q", user.Email, email)
	}
	if user.Name != name {
		t.Errorf("Name = %q, want %q", user.Name, name)
	}
	if user.CreatedAt == "" {
		t.Error("CreatedAt should be set, got empty string")
	}

	// Verify bcrypt hash matches the password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		t.Errorf("Bcrypt hash does not match password: %v", err)
	}
}

func TestSeedUser_SkipsIfUserAlreadyExists(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	email := "existing@example.com"
	// Pre-populate the store
	us.users[email] = &models.User{
		Email:        email,
		PasswordHash: "existing-hash",
		Name:         "Existing User",
		CreatedAt:    "2024-01-01T00:00:00Z",
	}

	// SeedUser should not return an error when user already exists
	err := seeder.SeedUser(email, "NewPassword!", "New Name")
	if err != nil {
		t.Fatalf("SeedUser returned error for existing user: %v", err)
	}

	// Verify the existing user was NOT overwritten
	user, _ := us.GetUser(email)
	if user.PasswordHash != "existing-hash" {
		t.Error("Existing user was overwritten — should have been skipped")
	}
	if user.Name != "Existing User" {
		t.Error("Existing user name was overwritten — should have been skipped")
	}
}

func TestSeedUser_BcryptHashVerification(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	password := "Docs4President!"
	err := seeder.SeedUser("user@example.com", password, "User")
	if err != nil {
		t.Fatalf("SeedUser returned error: %v", err)
	}

	user, _ := us.GetUser("user@example.com")

	// Verify correct password matches
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		t.Error("Correct password should match the hash")
	}

	// Verify wrong password does NOT match
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("WrongPassword"))
	if err == nil {
		t.Error("Wrong password should not match the hash")
	}
}

func TestSeedCustomers_CreatesAllCustomers(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	customers := []models.Customer{
		{ID: "cust-1", Name: "John Smith", Business: "Smith & Associates LLC"},
		{ID: "cust-2", Name: "Jane Doe", Business: "Doe Enterprises"},
		{ID: "cust-3", Name: "Robert Johnson", Business: "Johnson Construction Co."},
	}

	err := seeder.SeedCustomers(customers)
	if err != nil {
		t.Fatalf("SeedCustomers returned error: %v", err)
	}

	// Verify all customers were created
	for _, c := range customers {
		stored, err := cs.GetCustomer(c.ID)
		if err != nil {
			t.Fatalf("GetCustomer(%q) returned error: %v", c.ID, err)
		}
		if stored == nil {
			t.Errorf("Customer %q was not created", c.ID)
			continue
		}
		if stored.Name != c.Name {
			t.Errorf("Customer %q: Name = %q, want %q", c.ID, stored.Name, c.Name)
		}
		if stored.Business != c.Business {
			t.Errorf("Customer %q: Business = %q, want %q", c.ID, stored.Business, c.Business)
		}
	}
}

func TestSeedCustomers_SkipsExistingCustomers(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	// Pre-populate one customer
	cs.customers["cust-1"] = &models.Customer{
		ID:       "cust-1",
		Name:     "Original Name",
		Business: "Original Business",
	}

	customers := []models.Customer{
		{ID: "cust-1", Name: "John Smith", Business: "Smith & Associates LLC"},
		{ID: "cust-2", Name: "Jane Doe", Business: "Doe Enterprises"},
	}

	err := seeder.SeedCustomers(customers)
	if err != nil {
		t.Fatalf("SeedCustomers returned error: %v", err)
	}

	// cust-1 should NOT be overwritten
	stored1, _ := cs.GetCustomer("cust-1")
	if stored1.Name != "Original Name" {
		t.Error("Existing customer was overwritten — should have been skipped")
	}

	// cust-2 should be created
	stored2, _ := cs.GetCustomer("cust-2")
	if stored2 == nil {
		t.Fatal("New customer cust-2 was not created")
	}
	if stored2.Name != "Jane Doe" {
		t.Errorf("Customer cust-2: Name = %q, want %q", stored2.Name, "Jane Doe")
	}
}

func TestSeedSettings_CreatesDefaultFieldDefinitions(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	fields := []models.FieldDefinition{
		{Name: "Name", Type: "text"},
		{Name: "Business", Type: "text"},
		{Name: "Address", Type: "text"},
	}

	err := seeder.SeedSettings(fields)
	if err != nil {
		t.Fatalf("SeedSettings returned error: %v", err)
	}

	// Verify setting was created
	setting, err := ss.GetSetting("standard_fields")
	if err != nil {
		t.Fatalf("GetSetting returned error: %v", err)
	}
	if setting == nil {
		t.Fatal("standard_fields setting was not created")
	}

	// Verify the JSON value contains our fields
	var stored []models.FieldDefinition
	err = json.Unmarshal([]byte(setting.Value), &stored)
	if err != nil {
		t.Fatalf("Failed to unmarshal setting value: %v", err)
	}
	if len(stored) != len(fields) {
		t.Fatalf("Got %d fields, want %d", len(stored), len(fields))
	}
	for i, f := range stored {
		if f.Name != fields[i].Name {
			t.Errorf("Field %d: Name = %q, want %q", i, f.Name, fields[i].Name)
		}
		if f.Type != fields[i].Type {
			t.Errorf("Field %d: Type = %q, want %q", i, f.Type, fields[i].Type)
		}
	}
}

func TestSeedSettings_SkipsIfSettingsAlreadyExist(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	// Pre-populate the setting
	ss.settings["standard_fields"] = &models.Setting{
		Key:   "standard_fields",
		Value: `[{"name":"ExistingField","type":"text"}]`,
	}

	fields := []models.FieldDefinition{
		{Name: "Name", Type: "text"},
		{Name: "Business", Type: "text"},
	}

	err := seeder.SeedSettings(fields)
	if err != nil {
		t.Fatalf("SeedSettings returned error for existing settings: %v", err)
	}

	// Verify the existing setting was NOT overwritten
	setting, _ := ss.GetSetting("standard_fields")
	if setting.Value != `[{"name":"ExistingField","type":"text"}]` {
		t.Error("Existing settings were overwritten — should have been skipped")
	}
}

func TestSeedAll_RunsAllThreeSeeders(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	err := seeder.SeedAll("admin@example.com", "TestPass123!", "Admin User")
	if err != nil {
		t.Fatalf("SeedAll returned error: %v", err)
	}

	// Verify user was created
	user, _ := us.GetUser("admin@example.com")
	if user == nil {
		t.Error("SeedAll did not create the user")
	}
	if user != nil && user.Name != "Admin User" {
		t.Errorf("User Name = %q, want %q", user.Name, "Admin User")
	}

	// Verify customers were created (SeedAll should create 5 default customers)
	customers, _ := cs.ListCustomers()
	if len(customers) != 5 {
		t.Errorf("SeedAll created %d customers, want 5", len(customers))
	}

	// Verify settings were created
	setting, _ := ss.GetSetting("standard_fields")
	if setting == nil {
		t.Error("SeedAll did not create standard_fields setting")
	}

	// Verify the standard fields contain the expected 7 fields
	if setting != nil {
		var fields []models.FieldDefinition
		err = json.Unmarshal([]byte(setting.Value), &fields)
		if err != nil {
			t.Fatalf("Failed to unmarshal standard_fields: %v", err)
		}
		if len(fields) != 7 {
			t.Errorf("standard_fields has %d fields, want 7", len(fields))
		}
	}
}

func TestSeedAll_DefaultCustomerData(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	err := seeder.SeedAll("admin@example.com", "pass", "Admin")
	if err != nil {
		t.Fatalf("SeedAll returned error: %v", err)
	}

	// Verify the specific customer data
	customers, _ := cs.ListCustomers()
	nameSet := make(map[string]bool)
	for _, c := range customers {
		nameSet[c.Name] = true
	}

	expectedNames := []string{
		"John Smith",
		"Jane Doe",
		"Robert Johnson",
		"Maria Garcia",
		"David Williams",
	}
	for _, name := range expectedNames {
		if !nameSet[name] {
			t.Errorf("Expected customer %q not found in seeded data", name)
		}
	}
}

func TestSeedAll_DefaultFieldDefinitions(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	err := seeder.SeedAll("admin@example.com", "pass", "Admin")
	if err != nil {
		t.Fatalf("SeedAll returned error: %v", err)
	}

	setting, _ := ss.GetSetting("standard_fields")
	if setting == nil {
		t.Fatal("standard_fields not created")
	}

	var fields []models.FieldDefinition
	err = json.Unmarshal([]byte(setting.Value), &fields)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	expectedNames := []string{"Name", "Business", "Address", "City", "State", "Zip", "Phone Number"}
	if len(fields) != len(expectedNames) {
		t.Fatalf("Got %d fields, want %d", len(fields), len(expectedNames))
	}
	for i, f := range fields {
		if f.Name != expectedNames[i] {
			t.Errorf("Field %d: Name = %q, want %q", i, f.Name, expectedNames[i])
		}
		if f.Type != "text" {
			t.Errorf("Field %d: Type = %q, want %q", i, f.Type, "text")
		}
	}
}

func TestSeedAll_IsIdempotent(t *testing.T) {
	us := NewMockUserStore()
	cs := NewMockCustomerStore()
	ss := NewMockSettingsStore()
	seeder := NewSeeder(us, cs, ss)

	// Run SeedAll twice — second call should not error
	err := seeder.SeedAll("admin@example.com", "pass", "Admin")
	if err != nil {
		t.Fatalf("First SeedAll returned error: %v", err)
	}

	err = seeder.SeedAll("admin@example.com", "pass", "Admin")
	if err != nil {
		t.Fatalf("Second SeedAll returned error (not idempotent): %v", err)
	}

	// Verify only 5 customers exist (not 10)
	customers, _ := cs.ListCustomers()
	if len(customers) != 5 {
		t.Errorf("After two SeedAll calls, got %d customers, want 5", len(customers))
	}
}
