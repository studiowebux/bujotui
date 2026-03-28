package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	cfg, err := Load(dir, dataDir)
	if err != nil {
		t.Fatalf("Load with no file: %v", err)
	}

	if cfg.Dir != dir {
		t.Errorf("Dir = %q, want %q", cfg.Dir, dir)
	}
	if cfg.DataDir != dataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, dataDir)
	}
	if cfg.Symbols == nil {
		t.Fatal("Symbols is nil")
	}
	if len(cfg.Symbols.Symbols) == 0 {
		t.Error("expected default symbols to be populated")
	}
	// Check a known default symbol exists
	if _, ok := cfg.Symbols.Symbols["task"]; !ok {
		t.Error("expected default symbol 'task'")
	}
	if cfg.Defaults.Project != "inbox" {
		t.Errorf("Defaults.Project = %q, want %q", cfg.Defaults.Project, "inbox")
	}
	if cfg.Defaults.Person != "self" {
		t.Errorf("Defaults.Person = %q, want %q", cfg.Defaults.Person, "self")
	}
	if cfg.Defaults.Symbol != "task" {
		t.Errorf("Defaults.Symbol = %q, want %q", cfg.Defaults.Symbol, "task")
	}
}

func TestLoadValidConfigFile(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	conf := `[symbols]
task = .
done = x
note = -

[transitions]
task = done
done =
note =

[projects]
myproject

[people]
alice

[colors]
done = green

[defaults]
project = myproject
person = alice
symbol = task
`
	if err := os.WriteFile(filepath.Join(dir, "bujotui.conf"), []byte(conf), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(dir, dataDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Symbols.Symbols) != 3 {
		t.Errorf("got %d symbols, want 3", len(cfg.Symbols.Symbols))
	}
	if cfg.Symbols.Symbols["task"].Char != "." {
		t.Errorf("task char = %q, want %q", cfg.Symbols.Symbols["task"].Char, ".")
	}
	if cfg.Projects[0] != "myproject" {
		t.Errorf("Projects[0] = %q, want %q", cfg.Projects[0], "myproject")
	}
	if cfg.People[0] != "alice" {
		t.Errorf("People[0] = %q, want %q", cfg.People[0], "alice")
	}
	if cfg.Defaults.Project != "myproject" {
		t.Errorf("Defaults.Project = %q, want %q", cfg.Defaults.Project, "myproject")
	}
	if cfg.Defaults.Person != "alice" {
		t.Errorf("Defaults.Person = %q, want %q", cfg.Defaults.Person, "alice")
	}
	if cfg.Colors["done"] != "green" {
		t.Errorf("Colors[done] = %q, want %q", cfg.Colors["done"], "green")
	}
}

func TestValidation_DuplicateSymbols(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	// The SymbolSet.Add overwrites, but validate checks Order for dupes.
	// We need to craft a config that produces duplicate Order entries.
	// Since parseReader calls cfg.Symbols.Add which appends to Order each time,
	// having two lines with the same name creates a duplicate in Order.
	conf := `[symbols]
task = .
task = x

[transitions]

[defaults]
symbol = task
`
	if err := os.WriteFile(filepath.Join(dir, "bujotui.conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir, dataDir)
	if err == nil {
		t.Fatal("expected error for duplicate symbols")
	}
	if got := err.Error(); !contains(got, "duplicate symbol") {
		t.Errorf("error = %q, want it to contain 'duplicate symbol'", got)
	}
}

func TestValidation_InvalidTransitionTarget(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	conf := `[symbols]
task = .
done = x

[transitions]
task = done, nonexistent

[defaults]
symbol = task
`
	if err := os.WriteFile(filepath.Join(dir, "bujotui.conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir, dataDir)
	if err == nil {
		t.Fatal("expected error for invalid transition target")
	}
	if got := err.Error(); !contains(got, "transition target") {
		t.Errorf("error = %q, want it to contain 'transition target'", got)
	}
}

func TestValidation_InvalidColor(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	conf := `[symbols]
task = .

[transitions]
task =

[colors]
task = neon_pink

[defaults]
symbol = task
`
	if err := os.WriteFile(filepath.Join(dir, "bujotui.conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir, dataDir)
	if err == nil {
		t.Fatal("expected error for invalid color")
	}
	if got := err.Error(); !contains(got, "unknown color") {
		t.Errorf("error = %q, want it to contain 'unknown color'", got)
	}
}

func TestValidation_UnknownDefaultSymbol(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	conf := `[symbols]
task = .

[transitions]
task =

[defaults]
symbol = bogus
`
	if err := os.WriteFile(filepath.Join(dir, "bujotui.conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir, dataDir)
	if err == nil {
		t.Fatal("expected error for unknown default symbol")
	}
	if got := err.Error(); !contains(got, "default symbol") {
		t.Errorf("error = %q, want it to contain 'default symbol'", got)
	}
}

func TestDefaultConfigDir_NonEmpty(t *testing.T) {
	dir := DefaultConfigDir()
	if dir == "" {
		t.Error("DefaultConfigDir returned empty string")
	}
}

func TestDefaultDataDir_NonEmpty(t *testing.T) {
	dir := DefaultDataDir()
	if dir == "" {
		t.Error("DefaultDataDir returned empty string")
	}
}

func TestDefaultConfigDir_EnvOverride(t *testing.T) {
	t.Setenv("BUJOTUI_CONFIG_DIR", "/tmp/custom-config")
	dir := DefaultConfigDir()
	if dir != "/tmp/custom-config" {
		t.Errorf("DefaultConfigDir = %q, want /tmp/custom-config", dir)
	}
}

func TestDefaultDataDir_EnvOverride(t *testing.T) {
	t.Setenv("BUJOTUI_DATA_DIR", "/tmp/custom-data")
	dir := DefaultDataDir()
	if dir != "/tmp/custom-data" {
		t.Errorf("DefaultDataDir = %q, want /tmp/custom-data", dir)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
