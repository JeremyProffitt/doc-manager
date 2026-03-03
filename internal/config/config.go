package config

import "os"

// Config holds all application configuration values, loaded from
// environment variables with sensible defaults for local development.
type Config struct {
	AWSRegion      string
	S3Bucket       string
	BedrockModelID string
	JWTSecret      string

	// DynamoDB table names
	UsersTable           string
	FormsTable           string
	FieldPlacementsTable string
	CustomersTable       string
	SettingsTable        string
	SessionsTable        string
}

// Load reads configuration from environment variables, falling back to
// defaults when a variable is unset or empty.
func Load() *Config {
	return &Config{
		AWSRegion:            getEnv("AWS_REGION_NAME", "us-east-1"),
		S3Bucket:             getEnv("S3_BUCKET", "doc-manager-forms"),
		BedrockModelID:       getEnv("BEDROCK_MODEL_ID", "anthropic.claude-sonnet-4-20250514"),
		JWTSecret:            getEnv("JWT_SECRET", "dev-secret-change-me"),
		UsersTable:           getEnv("USERS_TABLE", "DocMgr-Users"),
		FormsTable:           getEnv("FORMS_TABLE", "DocMgr-Forms"),
		FieldPlacementsTable: getEnv("FIELD_PLACEMENTS_TABLE", "DocMgr-FieldPlacements"),
		CustomersTable:       getEnv("CUSTOMERS_TABLE", "DocMgr-Customers"),
		SettingsTable:        getEnv("SETTINGS_TABLE", "DocMgr-Settings"),
		SessionsTable:        getEnv("SESSIONS_TABLE", "DocMgr-Sessions"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
