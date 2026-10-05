package persona_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NubiMa/hikari-cli/internal/persona"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	content := `
name: TestBot
description: A test persona
greeting: "Hello from TestBot"
system_prompt: "You are TestBot."
behavior:
  tone: casual
  language: English
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := persona.LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if p.Name != "TestBot" {
		t.Errorf("expected 'TestBot', got %q", p.Name)
	}
	if p.Greeting != "Hello from TestBot" {
		t.Errorf("wrong greeting: %q", p.Greeting)
	}
}

func TestManagerDefault(t *testing.T) {
	// Empty builtin + user dirs → should get default fallback persona.
	builtin := t.TempDir()
	user := t.TempDir()

	mgr, err := persona.NewManagerFromDirs(builtin, user)
	if err != nil {
		t.Fatalf("NewManagerFromDirs: %v", err)
	}

	active := mgr.Active()
	if active == nil {
		t.Fatal("expected non-nil active persona")
	}
	if active.Name == "" {
		t.Error("expected non-empty persona name")
	}
}

func TestManagerSwitch(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "alpha.yaml", "Alpha", "First persona")
	writeYAML(t, dir, "beta.yaml", "Beta", "Second persona")

	mgr, err := persona.NewManagerFromDirs(dir, t.TempDir())
	if err != nil {
		t.Fatalf("NewManagerFromDirs: %v", err)
	}

	if err := mgr.SetActive("Alpha"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if mgr.Active().Name != "Alpha" {
		t.Errorf("expected Alpha, got %q", mgr.Active().Name)
	}

	if err := mgr.SetActive("Beta"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if mgr.Active().Name != "Beta" {
		t.Errorf("expected Beta, got %q", mgr.Active().Name)
	}

	if err := mgr.SetActive("nonexistent"); err == nil {
		t.Error("expected error for unknown persona")
	}
}

func TestManagerList(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "alpha.yaml", "Alpha", "")
	writeYAML(t, dir, "beta.yaml", "Beta", "")
	writeYAML(t, dir, "gamma.yaml", "Gamma", "")

	mgr, err := persona.NewManagerFromDirs(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	names := mgr.List()
	if len(names) != 3 {
		t.Errorf("expected 3 personas, got %d: %v", len(names), names)
	}
}

// TestManagerEmbedded verifies that the production NewManagerDefault loads
// at least one builtin persona from the embedded assets.
func TestManagerEmbedded(t *testing.T) {
	mgr, err := persona.NewManagerDefault()
	if err != nil {
		t.Fatalf("NewManagerDefault: %v", err)
	}
	all := mgr.All()
	if len(all) == 0 {
		t.Fatal("expected at least one persona from embedded assets")
	}
	// Hikari is a canonical builtin — verify it exists.
	_, found := mgr.Get("hikari")
	if !found {
		t.Error("expected builtin 'hikari' persona to be present")
	}
}

func writeYAML(t *testing.T, dir, filename, name, desc string) {
	t.Helper()
	content := "name: " + name + "\ndescription: " + desc + "\ngreeting: Hello\nsystem_prompt: You are " + name + ".\n"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
