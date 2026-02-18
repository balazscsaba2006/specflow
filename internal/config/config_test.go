package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsDefaultsWhenFileMissing(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "careful" {
		t.Errorf("expected mode 'careful', got %q", cfg.Mode)
	}
	if cfg.DefaultPriority != "medium" {
		t.Errorf("expected default_priority 'medium', got %q", cfg.DefaultPriority)
	}
}

func TestLoadReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("mode: fast\ndefault_priority: high\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != "fast" {
		t.Errorf("expected mode 'fast', got %q", cfg.Mode)
	}
	if cfg.DefaultPriority != "high" {
		t.Errorf("expected default_priority 'high', got %q", cfg.DefaultPriority)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{Mode: "fast", ConventionsFile: "README.md", DefaultPriority: "low"}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Mode != "fast" {
		t.Errorf("expected mode 'fast', got %q", loaded.Mode)
	}
	if loaded.ConventionsFile != "README.md" {
		t.Errorf("expected conventions_file 'README.md', got %q", loaded.ConventionsFile)
	}
}

func TestGlobalConfigDefaultMode(t *testing.T) {
	// LoadGlobal uses os.UserHomeDir(), so we test the parsing logic directly.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("default_mode: fast\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var gc GlobalConfig
	if err := parseGlobalConfig(data, &gc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gc.DefaultMode != "fast" {
		t.Errorf("expected default_mode 'fast', got %q", gc.DefaultMode)
	}
}

func TestDefaultConfigAppliesGlobalMode(t *testing.T) {
	// Simulate the init_cmd logic: DefaultConfig + global override.
	cfg := DefaultConfig()
	if cfg.Mode != "careful" {
		t.Fatalf("expected default mode 'careful', got %q", cfg.Mode)
	}

	// Apply global override (same logic as init_cmd.go).
	globalMode := "fast"
	if globalMode == "careful" || globalMode == "fast" {
		cfg.Mode = globalMode
	}

	if cfg.Mode != "fast" {
		t.Errorf("expected mode 'fast' after global override, got %q", cfg.Mode)
	}
}
