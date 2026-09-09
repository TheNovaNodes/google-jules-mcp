package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	content := "# A comment\nKEY_ONE=value_one\nKEY_TWO=\"value_two\"\nKEY_THREE='value_three'\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	if err := os.Unsetenv("KEY_ONE"); err != nil {
		t.Logf("err: %v", err)
	}
	if err := os.Unsetenv("KEY_TWO"); err != nil {
		t.Logf("err: %v", err)
	}
	if err := os.Unsetenv("KEY_THREE"); err != nil {
		t.Logf("err: %v", err)
	}

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

func TestLoadDotEnv_FileNotFound(t *testing.T) {
	loadDotEnv("nonexistent_file.env")
}

func TestLoadDotEnv_ExistingKeyNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	content := "KEY_FOUR=new_value\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	if err := os.Setenv("KEY_FOUR", "old_value"); err != nil {
		t.Logf("err: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("KEY_FOUR"); err != nil {
			t.Logf("err: %v", err)
		}
	}()

	loadDotEnv(envPath)

	if os.Getenv("KEY_FOUR") != "old_value" {
		t.Errorf("expected old_value, got %q", os.Getenv("KEY_FOUR"))
	}
}

func TestLoadDotEnv_EmptyLine(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	content := "KEY_FIVE=value_five\n\n\nKEY_SIX=value_six\nINVALID_LINE"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	if err := os.Unsetenv("KEY_FIVE"); err != nil {
		t.Logf("err: %v", err)
	}
	if err := os.Unsetenv("KEY_SIX"); err != nil {
		t.Logf("err: %v", err)
	}

	loadDotEnv(envPath)

	if os.Getenv("KEY_FIVE") != "value_five" {
		t.Errorf("expected value_five, got %q", os.Getenv("KEY_FIVE"))
	}
	if os.Getenv("KEY_SIX") != "value_six" {
		t.Errorf("expected value_six, got %q", os.Getenv("KEY_SIX"))
	}
}

func TestLoadDotEnv_DoubleQuotesUnterminated(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	content := "KEY_SEVEN=\"unclosed_quote"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	if err := os.Unsetenv("KEY_SEVEN"); err != nil {
		t.Logf("err: %v", err)
	}

	loadDotEnv(envPath)

	if os.Getenv("KEY_SEVEN") != "unclosed_quote" {
		t.Errorf("expected unclosed_quote, got %q", os.Getenv("KEY_SEVEN"))
	}
}
