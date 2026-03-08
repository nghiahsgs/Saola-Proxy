package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected log_level info, got %s", cfg.LogLevel)
	}
	if !cfg.AuditEnabled {
		t.Error("expected audit_enabled true")
	}
	if len(cfg.Whitelist) == 0 {
		t.Error("expected non-empty whitelist")
	}
}

func TestLoadFromPath_Valid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	content := `version: 1
log_level: debug
audit_enabled: false
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected debug, got %s", cfg.LogLevel)
	}
	if cfg.AuditEnabled {
		t.Error("expected audit_enabled false")
	}
}

func TestLoadFromPath_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yaml")
	if err := os.WriteFile(path, []byte(":\tinvalid:::yaml"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromPath(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_NoConfigFiles(t *testing.T) {
	// Point XDG to non-existent dir and ensure no cwd .saola.yaml exists.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "nonexistent"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config from defaults")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log_level info, got %s", cfg.LogLevel)
	}
}

func TestLoad_ProjectOverridesGlobal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Write global config.
	saolaDir := filepath.Join(tmp, "saola")
	if err := os.MkdirAll(saolaDir, 0700); err != nil {
		t.Fatal(err)
	}
	globalContent := `version: 1
log_level: warn
audit_enabled: true
`
	if err := os.WriteFile(filepath.Join(saolaDir, "config.yaml"), []byte(globalContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Write project config in a temp dir and change cwd.
	projectDir := t.TempDir()
	projectContent := `log_level: debug
`
	if err := os.WriteFile(filepath.Join(projectDir, ".saola.yaml"), []byte(projectContent), 0600); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Project overrides global for log_level.
	if cfg.LogLevel != "debug" {
		t.Errorf("expected debug from project config, got %s", cfg.LogLevel)
	}
	// Global value still present for audit_enabled.
	if !cfg.AuditEnabled {
		t.Error("expected audit_enabled true from global config")
	}
}

func TestConfigDir_WithXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/tmp/xdg/saola"
	if dir != want {
		t.Errorf("expected %s, got %s", want, dir)
	}
}

func TestConfigDir_WithoutXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("expected non-empty config dir")
	}
	// Should end with .saola
	base := filepath.Base(dir)
	if base != ".saola" {
		t.Errorf("expected .saola suffix, got %s", base)
	}
}

func TestWriteToFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.yaml")

	cfg := DefaultConfig()
	cfg.LogLevel = "debug"

	if err := cfg.WriteToFile(path); err != nil {
		t.Fatalf("WriteToFile error: %v", err)
	}

	loaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath error: %v", err)
	}
	if loaded.LogLevel != "debug" {
		t.Errorf("expected debug, got %s", loaded.LogLevel)
	}

	// Check file permissions.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 perms, got %o", info.Mode().Perm())
	}
}
