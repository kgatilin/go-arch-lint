package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/config"
)

func TestLoad_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .goarchlint
	configContent := `module: github.com/test/project
scan_paths:
  - cmd
  - pkg
ignore_paths:
  - vendor
rules:
  directories_import:
    cmd: [pkg, internal]
    pkg: [internal]
  detect_unused: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Module != "github.com/test/project" {
		t.Errorf("expected module github.com/test/project, got %s", cfg.Module)
	}

	if len(cfg.ScanPaths) != 2 {
		t.Errorf("expected 2 scan paths, got %d", len(cfg.ScanPaths))
	}

	if len(cfg.IgnorePaths) != 1 {
		t.Errorf("expected 1 ignore path, got %d", len(cfg.IgnorePaths))
	}

	if !cfg.Rules.DetectUnused {
		t.Error("expected detect_unused to be true")
	}
}

func TestLoad_WithoutConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only go.mod
	goMod := `module github.com/test/default

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should return default config
	if cfg.Module != "github.com/test/default" {
		t.Errorf("expected module github.com/test/default, got %s", cfg.Module)
	}

	if len(cfg.ScanPaths) != 3 {
		t.Errorf("expected 3 default scan paths, got %d", len(cfg.ScanPaths))
	}

	if !cfg.Rules.DetectUnused {
		t.Error("expected default detect_unused to be true")
	}
}

func TestLoad_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := config.Load(tmpDir)
	if err == nil {
		t.Fatal("expected error when go.mod is missing")
	}
}
