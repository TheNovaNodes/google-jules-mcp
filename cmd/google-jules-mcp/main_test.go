package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	content := `# A comment
KEY_ONE=value_one
KEY_TWO="value_two"
KEY_THREE='value_three'
`
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	_ = os.Unsetenv("KEY_ONE")
	_ = os.Unsetenv("KEY_TWO")
	_ = os.Unsetenv("KEY_THREE")

	loadDotEnv(envPath)

	if os.Getenv("KEY_ONE") != "value_one" {
		t.Errorf("expected value_one, got %q", os.Getenv("KEY_ONE"))
	}
	if os.Getenv("KEY_TWO") != "value_two" {
		t.Errorf("expected value_two, got %q", os.Getenv("KEY_TWO"))
	}
	if os.Getenv("KEY_THREE") != "value_three" {
		t.Errorf("expected value_three, got %q", os.Getenv("KEY_THREE"))
	}
}
