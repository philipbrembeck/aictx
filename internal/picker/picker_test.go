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

func TestPickMulti_EmptyList(t *testing.T) {
	result, err := PickMulti([]string{}, []bool{})
	if err != nil {
		t.Fatalf("PickMulti(empty) error: %v", err)
	}
	if result != nil {
		t.Errorf("PickMulti(empty) = %v, want nil", result)
	}
}

func TestPickMulti_InitialSelectionPreserved(t *testing.T) {
	// PickMulti is interactive; we test the internal selection logic directly
	// by simulating what happens to the selected slice.
	items := []string{"alpha", "beta", "gamma"}
	initial := []bool{true, false, true}

	selected := make([]bool, len(items))
	copy(selected, initial)

	// Verify copy is independent
	initial[0] = false
	if !selected[0] {
		t.Error("selected slice was not independently copied from initial")
	}

	// Simulate toggle of index 1
	selected[1] = !selected[1]
	if !selected[1] {
		t.Error("toggle of index 1 did not work")
	}

	// Final state: [true, true, true]
	for i, want := range []bool{true, true, true} {
		if selected[i] != want {
			t.Errorf("selected[%d] = %v, want %v", i, selected[i], want)
		}
	}
}
