package main

import (
	"log"
	"os"

	"github.com/JeremyProffitt/doc-manager/internal/config"
	"github.com/JeremyProffitt/doc-manager/internal/store"
)

func main() {
	cfg := config.Load()

	// Initialize DynamoDB client
	client, err := store.NewDynamoClient(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB client: %v", err)
	}

	// Create stores
	userStore := store.NewUserStore(client, cfg.UsersTable)
	customerStore := store.NewCustomerStore(client, cfg.CustomersTable)
	settingsStore := store.NewSettingsStore(client, cfg.SettingsTable)

	// Get password from env, fallback to default for local dev
	password := os.Getenv("SEED_USER_PASSWORD")
	if password == "" {
		password = "Docs4President!"
		log.Println("SEED_USER_PASSWORD not set, using default password")
	}

	// Run seeder
	seeder := store.NewSeeder(userStore, customerStore, settingsStore)
	err = seeder.SeedAll("proffitt.jeremy@gmail.com", password, "Jeremy Proffitt")
	if err != nil {
		log.Fatalf("Seed failed: %v", err)
	}

	log.Println("Seed completed successfully")
}
