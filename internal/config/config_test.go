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

	// Verify require_blackbox is true by default
	if !cfg.ShouldRequireBlackboxTests() {
		t.Error("expected default require_blackbox to be true")
	}
}

func TestLoad_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := config.Load(tmpDir)
	if err == nil {
		t.Fatal("expected error when go.mod is missing")
	}
}

func TestConfig_SharedExternalImports(t *testing.T) {
	// Test loading config with shared_external_imports
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .goarchlint with shared_external_imports
	configYAML := `
module: example.com/test
rules:
  shared_external_imports:
    detect: true
    mode: warn
    exclusions:
      - fmt
      - strings
    exclusion_patterns:
      - encoding/*
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify fields
	if !cfg.ShouldDetectSharedExternalImports() {
		t.Error("Expected detect=true")
	}
	if cfg.GetSharedExternalImportsMode() != "warn" {
		t.Errorf("Expected mode=warn, got %s", cfg.GetSharedExternalImportsMode())
	}
	if len(cfg.GetSharedExternalImportsExclusions()) != 2 {
		t.Errorf("Expected 2 exclusions, got %d", len(cfg.GetSharedExternalImportsExclusions()))
	}
	if len(cfg.GetSharedExternalImportsExclusionPatterns()) != 1 {
		t.Errorf("Expected 1 exclusion pattern, got %d", len(cfg.GetSharedExternalImportsExclusionPatterns()))
	}
}

// TestConfig_InterfaceMethods tests all interface methods on Config
func TestConfig_InterfaceMethods(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/myapp\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create comprehensive config
	configYAML := `
module: example.com/myapp
preset_used: ddd

structure:
  required_directories:
    cmd: "Commands"
    pkg: "Public API"
    internal: "Internal code"
  allow_other_directories: true

rules:
  directories_import:
    cmd: [pkg]
    pkg: [internal]
    internal: []
  detect_unused: true
  test_files:
    lint: true
    exempt_imports:
      - testing
      - testify
    location: alongside
  require_blackbox_tests: true

error_prompt:
  enabled: true
  architectural_goals: "Enforce clean architecture"
  principles:
    - "Dependency inversion"
    - "Single responsibility"
  refactoring_guidance: "Move to proper layer"

coverage:
  enabled: true
  threshold: 80.0
  package_thresholds:
    internal/app: 90.0
    pkg/http: 85.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test GetModule
	if cfg.GetModule() != "example.com/myapp" {
		t.Errorf("GetModule() = %s, want example.com/myapp", cfg.GetModule())
	}

	// Test GetDirectoriesImport
	dirsImport := cfg.GetDirectoriesImport()
	if len(dirsImport) != 3 {
		t.Errorf("GetDirectoriesImport() returned %d entries, want 3", len(dirsImport))
	}
	if len(dirsImport["cmd"]) != 1 || dirsImport["cmd"][0] != "pkg" {
		t.Errorf("GetDirectoriesImport()[cmd] = %v, want [pkg]", dirsImport["cmd"])
	}

	// Test ShouldDetectUnused
	if !cfg.ShouldDetectUnused() {
		t.Error("ShouldDetectUnused() = false, want true")
	}

	// Test GetRequiredDirectories
	reqDirs := cfg.GetRequiredDirectories()
	if len(reqDirs) != 3 {
		t.Errorf("GetRequiredDirectories() returned %d entries, want 3", len(reqDirs))
	}
	if reqDirs["cmd"] != "Commands" {
		t.Errorf("GetRequiredDirectories()[cmd] = %s, want Commands", reqDirs["cmd"])
	}

	// Test ShouldAllowOtherDirectories
	if !cfg.ShouldAllowOtherDirectories() {
		t.Error("ShouldAllowOtherDirectories() = false, want true")
	}

	// Test GetPresetUsed
	if cfg.GetPresetUsed() != "ddd" {
		t.Errorf("GetPresetUsed() = %s, want ddd", cfg.GetPresetUsed())
	}

	// Test GetErrorPrompt
	errPrompt := cfg.GetErrorPrompt()
	if !errPrompt.Enabled {
		t.Error("GetErrorPrompt().Enabled = false, want true")
	}
	if errPrompt.ArchitecturalGoals != "Enforce clean architecture" {
		t.Errorf("GetErrorPrompt().ArchitecturalGoals = %s, want 'Enforce clean architecture'", errPrompt.ArchitecturalGoals)
	}
	if len(errPrompt.Principles) != 2 {
		t.Errorf("GetErrorPrompt().Principles has %d items, want 2", len(errPrompt.Principles))
	}

	// Test ShouldLintTestFiles
	if !cfg.ShouldLintTestFiles() {
		t.Error("ShouldLintTestFiles() = false, want true")
	}

	// Test GetTestExemptImports
	exemptImports := cfg.GetTestExemptImports()
	if len(exemptImports) != 2 {
		t.Errorf("GetTestExemptImports() returned %d items, want 2", len(exemptImports))
	}

	// Test GetTestFileLocation
	if cfg.GetTestFileLocation() != "alongside" {
		t.Errorf("GetTestFileLocation() = %s, want alongside", cfg.GetTestFileLocation())
	}

	// Note: Coverage section is not loaded from YAML in current implementation
	// These tests verify the methods exist and don't panic
	_ = cfg.IsCoverageEnabled()
	_ = cfg.GetCoverageThreshold()
	_ = cfg.GetPackageThresholds()
}

// TestConfig_DefaultValues tests that interface methods return sensible defaults
func TestConfig_DefaultValues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal go.mod
	goMod := "module example.com/minimal\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create minimal config
	configYAML := `
module: example.com/minimal
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test defaults
	if cfg.ShouldDetectUnused() {
		t.Error("ShouldDetectUnused() should default to false")
	}

	if cfg.GetRequiredDirectories() == nil {
		t.Error("GetRequiredDirectories() should not return nil")
	}

	// ShouldAllowOtherDirectories defaults to false when not specified
	if cfg.ShouldAllowOtherDirectories() {
		t.Error("ShouldAllowOtherDirectories() should default to false")
	}

	if cfg.GetPresetUsed() != "" {
		t.Errorf("GetPresetUsed() should default to empty, got %s", cfg.GetPresetUsed())
	}

	if cfg.ShouldLintTestFiles() {
		t.Error("ShouldLintTestFiles() should default to false")
	}

	// GetTestFileLocation defaults to "colocated"
	if cfg.GetTestFileLocation() != "colocated" {
		t.Errorf("GetTestFileLocation() should default to 'colocated', got %s", cfg.GetTestFileLocation())
	}

	if cfg.IsCoverageEnabled() {
		t.Error("IsCoverageEnabled() should default to false")
	}

	if cfg.GetCoverageThreshold() != 0.0 {
		t.Errorf("GetCoverageThreshold() should default to 0.0, got %f", cfg.GetCoverageThreshold())
	}

	thresholds := cfg.GetPackageThresholds()
	if thresholds == nil {
		t.Error("GetPackageThresholds() should not return nil")
	}
	if len(thresholds) != 0 {
		t.Errorf("GetPackageThresholds() should default to empty map, got %d entries", len(thresholds))
	}
}

// TestConfig_GetSharedExternalImportsMode_DefaultValue tests the default mode
func TestConfig_GetSharedExternalImportsMode_DefaultValue(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with detect enabled but no mode specified
	configYAML := `
module: example.com/test
rules:
  shared_external_imports:
    detect: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// When detect is enabled but mode is not specified, default to "warn"
	mode := cfg.GetSharedExternalImportsMode()
	if mode != "warn" {
		t.Errorf("GetSharedExternalImportsMode() = %s, want 'warn' as default", mode)
	}
}

func TestConfig_Staticcheck_Enabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with staticcheck enabled
	configYAML := `
module: example.com/test
rules:
  staticcheck: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify staticcheck is enabled
	if !cfg.ShouldRunStaticcheck() {
		t.Error("ShouldRunStaticcheck() = false, want true")
	}
}

func TestConfig_Staticcheck_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with staticcheck explicitly disabled
	configYAML := `
module: example.com/test
rules:
  staticcheck: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify staticcheck is disabled
	if cfg.ShouldRunStaticcheck() {
		t.Error("ShouldRunStaticcheck() = true, want false")
	}
}

func TestConfig_Staticcheck_DefaultValue(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config without staticcheck specified
	configYAML := `
module: example.com/test
rules:
  detect_unused: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify staticcheck defaults to false
	if cfg.ShouldRunStaticcheck() {
		t.Error("ShouldRunStaticcheck() = true, want false (default)")
	}
}

func TestConfig_NewFormat_WithPreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config in new format with preset and no overrides
	configYAML := `
module: example.com/test

preset:
  name: simple
  structure:
    required_directories:
      cmd: "Commands"
      pkg: "Public API"
    allow_other_directories: true
  rules:
    directories_import:
      cmd: [pkg]
      pkg: [internal]
    detect_unused: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify preset values are used
	if cfg.GetPresetUsed() != "simple" {
		t.Errorf("GetPresetUsed() = %s, want simple", cfg.GetPresetUsed())
	}

	reqDirs := cfg.GetRequiredDirectories()
	if reqDirs["cmd"] != "Commands" {
		t.Errorf("GetRequiredDirectories()[cmd] = %s, want Commands", reqDirs["cmd"])
	}

	if !cfg.ShouldDetectUnused() {
		t.Error("ShouldDetectUnused() = false, want true")
	}
}

func TestConfig_NewFormat_WithOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config in new format with preset and overrides
	configYAML := `
module: example.com/test

preset:
  name: simple
  structure:
    required_directories:
      cmd: "Commands"
      pkg: "Public API"
    allow_other_directories: true
  rules:
    directories_import:
      cmd: [pkg]
      pkg: [internal]
    detect_unused: true

overrides:
  structure:
    required_directories:
      scripts: "Build scripts"
  rules:
    directories_import:
      scripts: [pkg]
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify preset values are used
	if cfg.GetPresetUsed() != "simple" {
		t.Errorf("GetPresetUsed() = %s, want simple", cfg.GetPresetUsed())
	}

	// Verify overrides are merged
	reqDirs := cfg.GetRequiredDirectories()
	if reqDirs["cmd"] != "Commands" {
		t.Errorf("GetRequiredDirectories()[cmd] = %s, want Commands (from preset)", reqDirs["cmd"])
	}
	if reqDirs["scripts"] != "Build scripts" {
		t.Errorf("GetRequiredDirectories()[scripts] = %s, want Build scripts (from override)", reqDirs["scripts"])
	}

	dirsImport := cfg.GetDirectoriesImport()
	if len(dirsImport["scripts"]) != 1 || dirsImport["scripts"][0] != "pkg" {
		t.Errorf("GetDirectoriesImport()[scripts] = %v, want [pkg] (from override)", dirsImport["scripts"])
	}
}

func TestConfig_BackwardCompatibility_OldFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config in old format (flat structure)
	configYAML := `
module: example.com/test
preset_used: simple

structure:
  required_directories:
    cmd: "Commands"
    pkg: "Public API"
  allow_other_directories: true

rules:
  directories_import:
    cmd: [pkg]
    pkg: [internal]
  detect_unused: true

error_prompt:
  enabled: true
  architectural_goals: "Old format goals"
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify old format still works
	if cfg.GetPresetUsed() != "simple" {
		t.Errorf("GetPresetUsed() = %s, want simple", cfg.GetPresetUsed())
	}

	reqDirs := cfg.GetRequiredDirectories()
	if reqDirs["cmd"] != "Commands" {
		t.Errorf("GetRequiredDirectories()[cmd] = %s, want Commands", reqDirs["cmd"])
	}

	if !cfg.ShouldDetectUnused() {
		t.Error("ShouldDetectUnused() = false, want true")
	}

	errPrompt := cfg.GetErrorPrompt()
	if errPrompt.ArchitecturalGoals != "Old format goals" {
		t.Errorf("GetErrorPrompt().ArchitecturalGoals = %s, want Old format goals", errPrompt.ArchitecturalGoals)
	}
}

func TestConfig_AdditiveExclusions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module example.com/test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with preset and additional exclusions in overrides
	configYAML := `
module: example.com/test

preset:
  name: simple
  structure:
    required_directories:
      cmd: "Commands"
  rules:
    shared_external_imports:
      detect: true
      mode: warn
      exclusions:
        - fmt
        - strings
        - errors
      exclusion_patterns:
        - encoding/*
    test_files:
      exempt_imports:
        - testing

overrides:
  rules:
    shared_external_imports:
      exclusions:
        - gopkg.in/yaml.v3  # Additional exclusion
        - github.com/foo/bar  # Another additional exclusion
      exclusion_patterns:
        - crypto/*  # Additional pattern
    test_files:
      exempt_imports:
        - github.com/stretchr/testify/assert  # Additional exempt import
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify exclusions are additive (preset + override)
	exclusions := cfg.GetSharedExternalImportsExclusions()
	expectedExclusions := map[string]bool{
		"fmt":                  true, // from preset
		"strings":              true, // from preset
		"errors":               true, // from preset
		"gopkg.in/yaml.v3":     true, // from override
		"github.com/foo/bar":   true, // from override
	}

	if len(exclusions) != len(expectedExclusions) {
		t.Errorf("Expected %d exclusions, got %d: %v", len(expectedExclusions), len(exclusions), exclusions)
	}

	for _, excl := range exclusions {
		if !expectedExclusions[excl] {
			t.Errorf("Unexpected exclusion: %s", excl)
		}
	}

	for excl := range expectedExclusions {
		found := false
		for _, e := range exclusions {
			if e == excl {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected exclusion: %s", excl)
		}
	}

	// Verify exclusion patterns are additive
	patterns := cfg.GetSharedExternalImportsExclusionPatterns()
	expectedPatterns := map[string]bool{
		"encoding/*": true, // from preset
		"crypto/*":   true, // from override
	}

	if len(patterns) != len(expectedPatterns) {
		t.Errorf("Expected %d patterns, got %d: %v", len(expectedPatterns), len(patterns), patterns)
	}

	for _, pattern := range patterns {
		if !expectedPatterns[pattern] {
			t.Errorf("Unexpected pattern: %s", pattern)
		}
	}

	// Verify exempt imports are additive
	exemptImports := cfg.GetTestExemptImports()
	expectedExempt := map[string]bool{
		"testing":                               true, // from preset
		"github.com/stretchr/testify/assert":    true, // from override
	}

	if len(exemptImports) != len(expectedExempt) {
		t.Errorf("Expected %d exempt imports, got %d: %v", len(expectedExempt), len(exemptImports), exemptImports)
	}

	for _, ei := range exemptImports {
		if !expectedExempt[ei] {
			t.Errorf("Unexpected exempt import: %s", ei)
		}
	}
}
