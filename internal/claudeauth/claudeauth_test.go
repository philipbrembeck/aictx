package claudeauth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRead_FromFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	creds := `{"accessToken":"sk-ant-test","refreshToken":"rt-test"}`
	if err := os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), []byte(creds), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := readFromFile()
	if err != nil {
		t.Fatalf("readFromFile() error: %v", err)
	}
	if got != creds {
		t.Errorf("readFromFile() = %q, want %q", got, creds)
	}
}

func TestRead_FromFile_NotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	_, err := readFromFile()
	if err == nil {
		t.Error("readFromFile() expected error when file missing")
	}
}

func TestRead_RespectsClaudeConfigDir(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", customDir)

	creds := `{"accessToken":"custom-dir-token"}`
	if err := os.WriteFile(filepath.Join(customDir, ".credentials.json"), []byte(creds), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := readFromFile()
	if err != nil {
		t.Fatalf("readFromFile() error: %v", err)
	}
	if got != creds {
		t.Errorf("readFromFile() = %q, want %q", got, creds)
	}
}

func TestWriteToFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	creds := `{"accessToken":"write-test"}`
	if err := writeToFile(creds); err != nil {
		t.Fatalf("writeToFile() error: %v", err)
	}

	// Verify file exists with correct permissions.
	info, err := os.Stat(filepath.Join(claudeDir, ".credentials.json"))
	if err != nil {
		t.Fatalf("stat .credentials.json: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions = %o, want 0600", info.Mode().Perm())
	}

	// Verify content.
	got, err := readFromFile()
	if err != nil {
		t.Fatalf("readFromFile() after write error: %v", err)
	}
	if got != creds {
		t.Errorf("readFromFile() = %q, want %q", got, creds)
	}
}

func TestRemoveFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write then remove.
	creds := `{"accessToken":"remove-test"}`
	if err := writeToFile(creds); err != nil {
		t.Fatal(err)
	}

	if err := removeFile(); err != nil {
		t.Fatalf("removeFile() error: %v", err)
	}

	// Verify file is gone.
	if _, err := os.Stat(credentialsFilePath()); !os.IsNotExist(err) {
		t.Error("expected .credentials.json to be removed")
	}

	// Remove on already-missing is no-op.
	if err := removeFile(); err != nil {
		t.Errorf("removeFile() on missing file: %v", err)
	}
}
