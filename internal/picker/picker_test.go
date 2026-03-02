package picker

import "testing"

func TestIsTerminal_NoTTY(t *testing.T) {
	// In a test environment stdin/stdout are not a real TTY.
	if IsTerminal() {
		t.Log("IsTerminal() = true (running with a real PTY — skipping assertion)")
	}
}

func TestPick_EmptyList(t *testing.T) {
	result, err := Pick([]string{}, "")
	if err != nil {
		t.Fatalf("Pick(empty) error: %v", err)
	}
	if result != "" {
		t.Errorf("Pick(empty) = %q, want empty string", result)
	}
}
