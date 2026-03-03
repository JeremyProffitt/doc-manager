package store

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// bcryptCost is the cost factor for bcrypt hashing.
const bcryptCost = 12

// Seeder handles populating the data stores with initial seed data.
type Seeder struct {
	userStore     UserStore
	customerStore CustomerStore
	settingsStore SettingsStore
}

// NewSeeder creates a new Seeder with the given store implementations.
func NewSeeder(us UserStore, cs CustomerStore, ss SettingsStore) *Seeder {
	return &Seeder{
		userStore:     us,
		customerStore: cs,
		settingsStore: ss,
	}
}

// SeedUser creates a user with the given email, password, and name.
// The password is hashed with bcrypt before storage.
// If the user already exists, it is skipped gracefully.
func (s *Seeder) SeedUser(email, password, name string) error {
	// Check if user already exists
	existing, err := s.userStore.GetUser(email)
	if err != nil {
		return fmt.Errorf("checking existing user: %w", err)
	}
	if existing != nil {
		log.Printf("User %s already exists, skipping", email)
		return nil
	}

	// Hash the password with bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	log.Printf("Creating user %s...", email)
	if err := s.userStore.CreateUser(user); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	log.Printf("User %s created successfully", email)
	return nil
}

// SeedCustomers creates the given customers in the store.
// Existing customers (by ID) are skipped gracefully.
func (s *Seeder) SeedCustomers(customers []models.Customer) error {
	for _, c := range customers {
		existing, err := s.customerStore.GetCustomer(c.ID)
		if err != nil {
			return fmt.Errorf("checking existing customer %s: %w", c.ID, err)
		}
		if existing != nil {
			log.Printf("Customer %s (%s) already exists, skipping", c.ID, c.Name)
			continue
		}

		log.Printf("Creating customer %s (%s)...", c.ID, c.Name)
		if err := s.customerStore.CreateCustomer(&c); err != nil {
			return fmt.Errorf("creating customer %s: %w", c.ID, err)
		}
		log.Printf("Customer %s created successfully", c.Name)
	}
	return nil
}

// SeedSettings creates the "standard_fields" setting with the given field definitions.
// If the setting already exists, it is skipped gracefully.
func (s *Seeder) SeedSettings(fields []models.FieldDefinition) error {
	existing, err := s.settingsStore.GetSetting("standard_fields")
	if err != nil {
		return fmt.Errorf("checking existing settings: %w", err)
	}
	if existing != nil {
		log.Printf("Setting standard_fields already exists, skipping")
		return nil
	}

	value, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("marshaling field definitions: %w", err)
	}

	setting := &models.Setting{
		Key:   "standard_fields",
		Value: string(value),
	}

	log.Printf("Creating standard_fields setting with %d fields...", len(fields))
	if err := s.settingsStore.PutSetting(setting); err != nil {
		return fmt.Errorf("creating standard_fields setting: %w", err)
	}
	log.Printf("Standard fields setting created successfully")
	return nil
}

// defaultCustomers returns the 5 default seed customers.
func defaultCustomers() []models.Customer {
	return []models.Customer{
		{
			ID:       "seed-customer-001",
			Name:     "John Smith",
			Business: "Smith & Associates LLC",
			Address:  "123 Main Street",
			City:     "Austin",
			State:    "TX",
			Zip:      "78701",
			Phone:    "(512) 555-0101",
		},
		{
			ID:       "seed-customer-002",
			Name:     "Jane Doe",
			Business: "Doe Enterprises",
			Address:  "456 Oak Avenue",
			City:     "Dallas",
			State:    "TX",
			Zip:      "75201",
			Phone:    "(214) 555-0202",
		},
		{
			ID:       "seed-customer-003",
			Name:     "Robert Johnson",
			Business: "Johnson Construction Co.",
			Address:  "789 Pine Road",
			City:     "Houston",
			State:    "TX",
			Zip:      "77001",
			Phone:    "(713) 555-0303",
		},
		{
			ID:       "seed-customer-004",
			Name:     "Maria Garcia",
			Business: "Garcia Legal Services",
			Address:  "321 Elm Boulevard",
			City:     "San Antonio",
			State:    "TX",
			Zip:      "78201",
			Phone:    "(210) 555-0404",
		},
		{
			ID:       "seed-customer-005",
			Name:     "David Williams",
			Business: "Williams Financial Group",
			Address:  "654 Cedar Lane",
			City:     "Fort Worth",
			State:    "TX",
			Zip:      "76101",
			Phone:    "(817) 555-0505",
		},
	}
}

// defaultFieldDefinitions returns the 7 standard field definitions.
func defaultFieldDefinitions() []models.FieldDefinition {
	return []models.FieldDefinition{
		{Name: "Name", Type: "text"},
		{Name: "Business", Type: "text"},
		{Name: "Address", Type: "text"},
		{Name: "City", Type: "text"},
		{Name: "State", Type: "text"},
		{Name: "Zip", Type: "text"},
		{Name: "Phone Number", Type: "text"},
	}
}

// SeedAll runs all three seed operations: user, customers, and settings.
// It uses the default customer data and field definitions.
func (s *Seeder) SeedAll(email, password, name string) error {
	log.Println("Starting seed operation...")

	if err := s.SeedUser(email, password, name); err != nil {
		return fmt.Errorf("seeding user: %w", err)
	}

	if err := s.SeedCustomers(defaultCustomers()); err != nil {
		return fmt.Errorf("seeding customers: %w", err)
	}

	if err := s.SeedSettings(defaultFieldDefinitions()); err != nil {
		return fmt.Errorf("seeding settings: %w", err)
	}

	log.Println("Seed operation completed successfully")
	return nil
}
