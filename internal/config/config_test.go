package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.AWSRegion != "us-east-1" {
		t.Errorf("expected default AWSRegion us-east-1, got %s", cfg.AWSRegion)
	}
	if cfg.UsersTable != "DocMgr-Users" {
		t.Errorf("expected default UsersTable DocMgr-Users, got %s", cfg.UsersTable)
	}
	if cfg.SessionsTable != "DocMgr-Sessions" {
		t.Errorf("expected default SessionsTable DocMgr-Sessions, got %s", cfg.SessionsTable)
	}
	if cfg.BedrockModelID != "anthropic.claude-sonnet-4-20250514" {
		t.Errorf("expected default BedrockModelID, got %s", cfg.BedrockModelID)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("AWS_REGION_NAME", "eu-west-1")
	os.Setenv("S3_BUCKET", "my-bucket")
	os.Setenv("JWT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("AWS_REGION_NAME")
		os.Unsetenv("S3_BUCKET")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg := Load()

	if cfg.AWSRegion != "eu-west-1" {
		t.Errorf("expected AWSRegion eu-west-1, got %s", cfg.AWSRegion)
	}
	if cfg.S3Bucket != "my-bucket" {
		t.Errorf("expected S3Bucket my-bucket, got %s", cfg.S3Bucket)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Errorf("expected JWTSecret test-secret, got %s", cfg.JWTSecret)
	}
}
