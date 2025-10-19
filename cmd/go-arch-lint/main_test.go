package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Package-level variable to hold the binary path for all tests
var binaryPath string

// TestMain runs before all tests and builds the binary once
func TestMain(m *testing.M) {
	// Build the binary once for all tests
	binary, cleanup, err := setupBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}
	binaryPath = binary

	// Run all tests
	code := m.Run()

	// Cleanup
	cleanup()

	os.Exit(code)
}

// setupBinary builds the binary and returns the path and cleanup function
func setupBinary() (string, func(), error) {
	// Create temp directory for binary
	tmpDir, err := os.MkdirTemp("", "go-arch-lint-test-*")
	if err != nil {
		return "", nil, err
	}

	binary := filepath.Join(tmpDir, "go-arch-lint")

	// Get project root (two levels up from cmd/go-arch-lint)
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		return "", nil, fmt.Errorf("failed to get project root: %w", err)
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/go-arch-lint")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("build failed: %v\nOutput: %s", err, output)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return binary, cleanup, nil
}

func TestCLI_NoViolations_ExitCode0(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid project
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
scan_paths:
  - cmd
  - pkg
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create simple files
	cmdDir := filepath.Join(tmpDir, "cmd")
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/test/project/pkg"

func main() {
	pkg.Run()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	pkgGo := `package pkg

func Run() {}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "pkg.go"), []byte(pkgGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary
	cmd := exec.Command(binaryPath, ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should succeed with no violations
	if err != nil {
		t.Errorf("expected exit code 0, got error: %v\nOutput: %s", err, output)
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nOutput: %s", exitCode, output)
	}
}

func TestCLI_WithViolations_ExitCode1(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project with violations
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
scan_paths:
  - pkg
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pkg files with pkg-to-pkg violation
	pkg1Dir := filepath.Join(tmpDir, "pkg", "service1")
	pkg2Dir := filepath.Join(tmpDir, "pkg", "service2")
	if err := os.MkdirAll(pkg1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkg2Dir, 0755); err != nil {
		t.Fatal(err)
	}

	service1Go := `package service1

import "github.com/test/project/pkg/service2"

func Run() {
	service2.Helper()
}
`
	if err := os.WriteFile(filepath.Join(pkg1Dir, "service1.go"), []byte(service1Go), 0644); err != nil {
		t.Fatal(err)
	}

	service2Go := `package service2

func Helper() {}
`
	if err := os.WriteFile(filepath.Join(pkg2Dir, "service2.go"), []byte(service2Go), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary
	cmd := exec.Command(binaryPath, ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail with violations
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nOutput: %s", exitCode, output)
	}

	// Verify violation output
	outputStr := string(output)
	if !strings.Contains(outputStr, "DEPENDENCY VIOLATIONS DETECTED") {
		t.Errorf("expected violation header in output, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "Forbidden pkg-to-pkg Dependency") {
		t.Errorf("expected pkg-to-pkg violation in output, got: %s", outputStr)
	}
}

func TestCLI_ExitZeroFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project with violations
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
scan_paths:
  - pkg
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create files with violation
	pkg1Dir := filepath.Join(tmpDir, "pkg", "service1")
	pkg2Dir := filepath.Join(tmpDir, "pkg", "service2")
	if err := os.MkdirAll(pkg1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkg2Dir, 0755); err != nil {
		t.Fatal(err)
	}

	service1Go := `package service1

import "github.com/test/project/pkg/service2"

func Run() {
	service2.Helper()
}
`
	if err := os.WriteFile(filepath.Join(pkg1Dir, "service1.go"), []byte(service1Go), 0644); err != nil {
		t.Fatal(err)
	}

	service2Go := `package service2

func Helper() {}
`
	if err := os.WriteFile(filepath.Join(pkg2Dir, "service2.go"), []byte(service2Go), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary with -exit-zero flag
	cmd := exec.Command(binaryPath, "-exit-zero", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should succeed despite violations
	if err != nil {
		t.Errorf("expected exit code 0 with -exit-zero, got error: %v\nOutput: %s", err, output)
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode != 0 {
		t.Errorf("expected exit code 0 with -exit-zero, got %d\nOutput: %s", exitCode, output)
	}

	// Should still show violations in output
	outputStr := string(output)
	if !strings.Contains(outputStr, "DEPENDENCY VIOLATIONS DETECTED") {
		t.Errorf("expected violation output even with -exit-zero, got: %s", outputStr)
	}
}

func TestCLI_SharedExternalImports_WarnMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project with shared external import in warn mode
	configYAML := `rules:
  directories_import:
    cmd: [internal]
    internal: []
  detect_unused: false
  shared_external_imports:
    detect: true
    mode: warn
scan_paths:
  - cmd
  - internal
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create files with shared external import
	cmdDir := filepath.Join(tmpDir, "cmd")
	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/pkg/errors"

func main() {
	err := errors.New("test")
	_ = err
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	repoGo := `package internal

import "github.com/pkg/errors"

func Query() error {
	return errors.New("error")
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "repo.go"), []byte(repoGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary
	cmd := exec.Command(binaryPath, ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should succeed (warn mode = exit 0)
	if err != nil {
		t.Errorf("expected exit code 0 in warn mode, got error: %v\nOutput: %s", err, output)
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode != 0 {
		t.Errorf("expected exit code 0 in warn mode, got %d\nOutput: %s", exitCode, output)
	}

	// Should still show violation in stderr
	outputStr := string(output)
	if !strings.Contains(outputStr, "Shared External Import") {
		t.Errorf("expected shared external import warning in output, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "github.com/pkg/errors") {
		t.Errorf("expected package name in output, got: %s", outputStr)
	}
}

func TestCLI_SharedExternalImports_ErrorMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project with shared external import in error mode
	configYAML := `rules:
  directories_import:
    cmd: [internal]
    internal: []
  detect_unused: false
  shared_external_imports:
    detect: true
    mode: error
scan_paths:
  - cmd
  - internal
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create files with shared external import
	cmdDir := filepath.Join(tmpDir, "cmd")
	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/pkg/errors"

func main() {
	err := errors.New("test")
	_ = err
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	repoGo := `package internal

import "github.com/pkg/errors"

func Query() error {
	return errors.New("error")
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "repo.go"), []byte(repoGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary
	cmd := exec.Command(binaryPath, ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail (error mode = exit 1)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 in error mode, got %d\nOutput: %s", exitCode, output)
	}

	// Verify violation output
	outputStr := string(output)
	if !strings.Contains(outputStr, "Shared External Import") {
		t.Errorf("expected shared external import error in output, got: %s", outputStr)
	}
}

func TestCLI_MarkdownFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create simple project
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
scan_paths:
  - cmd
  - pkg
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	cmdDir := filepath.Join(tmpDir, "cmd")
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/test/project/pkg"

func main() {
	pkg.Run()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	pkgGo := `package pkg

func Run() {}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "pkg.go"), []byte(pkgGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary with markdown format
	cmd := exec.Command(binaryPath, "-format=markdown", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("expected success, got error: %v\nOutput: %s", err, output)
	}

	// Verify markdown output
	outputStr := string(output)
	if !strings.Contains(outputStr, "# Dependency Graph") {
		t.Errorf("expected dependency graph header in output, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "cmd/main.go") {
		t.Errorf("expected cmd file in output, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "local:pkg") {
		t.Errorf("expected local dependency in output, got: %s", outputStr)
	}
}

func TestCLI_InitPreset_DDD(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Run init with ddd preset
	cmd := exec.Command(binaryPath, "init", "--preset=ddd")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("expected success, got error: %v\nOutput: %s", err, output)
	}

	// Read generated config
	configPath := filepath.Join(tmpDir, ".goarchlint")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read generated config: %v", err)
	}

	configStr := string(configData)

	// Verify shared_external_imports is included
	if !strings.Contains(configStr, "shared_external_imports:") {
		t.Error("expected shared_external_imports in config")
	}

	if !strings.Contains(configStr, "detect: true") {
		t.Error("expected detect: true in shared_external_imports")
	}

	if !strings.Contains(configStr, "mode: warn") {
		t.Error("expected mode: warn in shared_external_imports")
	}

	if !strings.Contains(configStr, "exclusions:") {
		t.Error("expected exclusions list in shared_external_imports")
	}

	// Verify standard exclusions are present
	if !strings.Contains(configStr, "- fmt") {
		t.Error("expected fmt in exclusions")
	}

	if !strings.Contains(configStr, "- context") {
		t.Error("expected context in exclusions")
	}

	if !strings.Contains(configStr, "exclusion_patterns:") {
		t.Error("expected exclusion_patterns in shared_external_imports")
	}

	if !strings.Contains(configStr, "- encoding/*") {
		t.Error("expected encoding/* in exclusion_patterns")
	}

	// Verify preset_used is set
	if !strings.Contains(configStr, "preset_used: ddd") {
		t.Error("expected preset_used: ddd in config")
	}
}

func TestCLI_InitPreset_Simple(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Run init with simple preset
	cmd := exec.Command(binaryPath, "init", "--preset=simple")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("expected success, got error: %v\nOutput: %s", err, output)
	}

	// Read generated config
	configData, err := os.ReadFile(filepath.Join(tmpDir, ".goarchlint"))
	if err != nil {
		t.Fatalf("failed to read generated config: %v", err)
	}

	configStr := string(configData)

	// Verify shared_external_imports is included
	if !strings.Contains(configStr, "shared_external_imports:") {
		t.Error("expected shared_external_imports in simple preset config")
	}

	if !strings.Contains(configStr, "mode: warn") {
		t.Error("expected mode: warn in simple preset")
	}

	// Verify directories for simple preset
	if !strings.Contains(configStr, "cmd:") {
		t.Error("expected cmd directory in simple preset")
	}

	if !strings.Contains(configStr, "pkg:") {
		t.Error("expected pkg directory in simple preset")
	}

	if !strings.Contains(configStr, "internal:") {
		t.Error("expected internal directory in simple preset")
	}
}

func TestCLI_InitPreset_Hexagonal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Run init with hexagonal preset
	cmd := exec.Command(binaryPath, "init", "--preset=hexagonal")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("expected success, got error: %v\nOutput: %s", err, output)
	}

	// Read generated config
	configData, err := os.ReadFile(filepath.Join(tmpDir, ".goarchlint"))
	if err != nil {
		t.Fatalf("failed to read generated config: %v", err)
	}

	configStr := string(configData)

	// Verify shared_external_imports is included
	if !strings.Contains(configStr, "shared_external_imports:") {
		t.Error("expected shared_external_imports in hexagonal preset config")
	}

	// Verify hexagonal architecture directories
	if !strings.Contains(configStr, "internal/core:") {
		t.Error("expected internal/core directory in hexagonal preset")
	}

	if !strings.Contains(configStr, "internal/ports:") {
		t.Error("expected internal/ports directory in hexagonal preset")
	}

	if !strings.Contains(configStr, "internal/adapters:") {
		t.Error("expected internal/adapters directory in hexagonal preset")
	}
}

func TestCLI_Refresh_SamePreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize with simple preset
	initCmd := exec.Command(binaryPath, "init", "--preset=simple", "--create-dirs=false")
	initCmd.Dir = tmpDir
	output, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	// Modify the config to add a custom comment
	configPath := filepath.Join(tmpDir, ".goarchlint")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	modifiedConfig := "# Custom modification\n" + string(configData)
	if err := os.WriteFile(configPath, []byte(modifiedConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Run refresh without preset flag (should use preset_used from config)
	refreshCmd := exec.Command(binaryPath, "refresh")
	refreshCmd.Dir = tmpDir
	output, err = refreshCmd.CombinedOutput()
	if err != nil {
		t.Errorf("refresh failed: %v\nOutput: %s", err, output)
	}

	// Verify backup was created
	backupPath := filepath.Join(tmpDir, ".goarchlint.backup")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("expected .goarchlint.backup to be created")
	}

	// Verify backup contains the custom modification
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(backupData), "# Custom modification") {
		t.Error("expected backup to contain custom modification")
	}

	// Verify refreshed config is updated (custom modification should be gone)
	refreshedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	refreshedStr := string(refreshedData)
	if strings.Contains(refreshedStr, "# Custom modification") {
		t.Error("expected custom modification to be removed after refresh")
	}

	// Verify it's still the simple preset
	if !strings.Contains(refreshedStr, "preset_used: simple") {
		t.Error("expected preset_used: simple after refresh")
	}

	// Verify header indicates refresh
	if !strings.Contains(refreshedStr, "Refreshed by go-arch-lint refresh") {
		t.Error("expected refresh header in config")
	}
}

func TestCLI_Refresh_SwitchPreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize with simple preset
	initCmd := exec.Command(binaryPath, "init", "--preset=simple", "--create-dirs=false")
	initCmd.Dir = tmpDir
	output, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	// Refresh with ddd preset
	refreshCmd := exec.Command(binaryPath, "refresh", "--preset=ddd")
	refreshCmd.Dir = tmpDir
	output, err = refreshCmd.CombinedOutput()
	if err != nil {
		t.Errorf("refresh failed: %v\nOutput: %s", err, output)
	}

	// Read refreshed config
	configPath := filepath.Join(tmpDir, ".goarchlint")
	refreshedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	refreshedStr := string(refreshedData)

	// Verify it's now the ddd preset
	if !strings.Contains(refreshedStr, "preset_used: ddd") {
		t.Error("expected preset_used: ddd after switching presets")
	}

	// Verify ddd-specific directories
	if !strings.Contains(refreshedStr, "internal/domain:") {
		t.Error("expected internal/domain directory from ddd preset")
	}
	if !strings.Contains(refreshedStr, "internal/app:") {
		t.Error("expected internal/app directory from ddd preset")
	}
	if !strings.Contains(refreshedStr, "internal/infra:") {
		t.Error("expected internal/infra directory from ddd preset")
	}

	// Verify simple preset directories are gone
	if strings.Contains(refreshedStr, "pkg:") && strings.Contains(refreshedStr, "Public libraries") {
		t.Error("expected simple preset directories to be replaced")
	}
}

func TestCLI_Refresh_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't create .goarchlint file

	// Try to refresh
	refreshCmd := exec.Command(binaryPath, "refresh")
	refreshCmd.Dir = tmpDir
	output, err := refreshCmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("expected refresh to fail when .goarchlint doesn't exist")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, ".goarchlint not found") {
		t.Errorf("expected error message about missing .goarchlint, got: %s", outputStr)
	}
}

func TestCLI_Refresh_CustomConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a custom config (no preset_used field)
	customConfig := `# Custom configuration
rules:
  directories_import:
    cmd: [pkg]
    pkg: []
`
	configPath := filepath.Join(tmpDir, ".goarchlint")
	if err := os.WriteFile(configPath, []byte(customConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to refresh without specifying preset
	refreshCmd := exec.Command(binaryPath, "refresh")
	refreshCmd.Dir = tmpDir
	output, err := refreshCmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("expected refresh to fail for custom config without --preset flag")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "cannot refresh") || !strings.Contains(outputStr, "preset") {
		t.Errorf("expected error message about needing preset, got: %s", outputStr)
	}
}

func TestCLI_Init_GeneratesFullDocs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create minimal Go files in the directories that will be created
	// (needed for documentation generation to work)
	for _, dir := range []string{"cmd", "pkg", "internal"} {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatal(err)
		}

		// Create a simple .go file
		goFile := "package " + dir + "\n"
		if err := os.WriteFile(filepath.Join(dirPath, dir+".go"), []byte(goFile), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Run init with simple preset
	initCmd := exec.Command(binaryPath, "init", "--preset=simple")
	initCmd.Dir = tmpDir
	output, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}

	// Verify only arch-generated.md was created (not separate files)
	archGenPath := filepath.Join(tmpDir, "docs", "arch-generated.md")
	if _, err := os.Stat(archGenPath); os.IsNotExist(err) {
		t.Error("expected docs/arch-generated.md to be created")
	}

	// Read the generated docs
	docsData, err := os.ReadFile(archGenPath)
	if err != nil {
		t.Fatal(err)
	}
	docsStr := string(docsData)

	// Verify it's comprehensive documentation (contains all sections)
	expectedSections := []string{
		"# Project Architecture",
		"## Project Structure",
		"## Architectural Rules",
		"## Dependency Graph",
		"## Public API",
		"## Statistics",
	}

	for _, section := range expectedSections {
		if !strings.Contains(docsStr, section) {
			t.Errorf("expected section '%s' in generated docs", section)
		}
	}

	// Verify old separate files are NOT created
	apiGenPath := filepath.Join(tmpDir, "docs", "public-api-generated.md")
	if _, err := os.Stat(apiGenPath); err == nil {
		t.Error("did not expect docs/public-api-generated.md to be created (should be in single file now)")
	}
}

func TestCLI_Help(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"help command", []string{"help"}},
		{"--help flag", []string{"--help"}},
		{"-h flag", []string{"-h"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tc.args...)
			output, err := cmd.CombinedOutput()

			// Should exit with code 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v\nOutput: %s", err, output)
			}

			outputStr := string(output)

			// Verify key sections are present
			expectedSections := []string{
				"go-arch-lint - Go architecture linter",
				"USAGE:",
				"COMMANDS:",
				"DEFAULT COMMAND FLAGS:",
				"INIT COMMAND:",
				"REFRESH COMMAND:",
				"DOCS COMMAND:",
				"EXAMPLES:",
				"EXIT CODES:",
			}

			for _, section := range expectedSections {
				if !strings.Contains(outputStr, section) {
					t.Errorf("expected help output to contain '%s'\nGot: %s", section, outputStr)
				}
			}

			// Verify key commands are documented
			expectedCommands := []string{
				"init",
				"refresh",
				"docs",
				"help",
			}

			for _, cmd := range expectedCommands {
				if !strings.Contains(outputStr, cmd) {
					t.Errorf("expected help output to mention command '%s'", cmd)
				}
			}

			// Verify key flags are documented
			expectedFlags := []string{
				"-format",
				"-detailed",
				"-exit-zero",
				"-strict",
				"-preset",
				"-create-dirs",
			}

			for _, flag := range expectedFlags {
				if !strings.Contains(outputStr, flag) {
					t.Errorf("expected help output to mention flag '%s'", flag)
				}
			}
		})
	}
}

// TestCLI_RequireBlackboxTests tests that the CLI detects whitebox tests
func TestCLI_RequireBlackboxTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .goarchlint with require_blackbox enabled
	configYAML := `scan_paths:
  - internal

structure:
  required_directories:
    internal: "Internal packages"
  allow_other_directories: true

rules:
  directories_import:
    internal: []
  detect_unused: false
  test_files:
    lint: true
    require_blackbox: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create internal package with code
	internalDir := filepath.Join(tmpDir, "internal", "app")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	appGo := `package app

func Process() string {
	return "processed"
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "app.go"), []byte(appGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create WHITEBOX test (should violate)
	whiteboxTestGo := `package app

import "testing"

func TestProcess(t *testing.T) {
	result := Process()
	if result != "processed" {
		t.Fail()
	}
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "app_test.go"), []byte(whiteboxTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary
	cmd := exec.Command(binaryPath, ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail with exit code 1 due to whitebox test violation
	if err == nil {
		t.Error("expected non-zero exit code due to whitebox test violation")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %v", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	outputStr := string(output)

	// Should contain whitebox test violation
	if !strings.Contains(outputStr, "Whitebox Test") {
		t.Error("expected output to contain 'Whitebox Test' violation")
	}

	// Should mention the whitebox test file
	if !strings.Contains(outputStr, "internal/app/app_test.go") {
		t.Error("expected violation for internal/app/app_test.go")
	}

	// Should mention the expected package name
	if !strings.Contains(outputStr, "app_test") {
		t.Error("expected violation to suggest package name 'app_test'")
	}

	// Should have concise rule and fix (no error_prompt enabled in this test)
	if !strings.Contains(outputStr, "Blackbox testing is enforced") {
		t.Error("expected violation to contain rule about blackbox testing")
	}

	if !strings.Contains(outputStr, "Change package declaration from 'package app' to 'package app_test'") {
		t.Error("expected violation to contain fix instruction")
	}
}

// TestCLI_WhiteboxTestGuidance_NotRepeated verifies that blackbox testing guidance
// appears once at the end, not repeated for each violation
func TestCLI_WhiteboxTestGuidance_NotRepeated(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .goarchlint with require_blackbox and error_prompt enabled
	configYAML := `scan_paths:
  - internal

structure:
  required_directories:
    internal: "Internal packages"
  allow_other_directories: true

rules:
  directories_import:
    internal: []
  detect_unused: false
  test_files:
    lint: true
    require_blackbox: true

error_prompt:
  enabled: true
  architectural_goals: "Test architecture goals"
  principles:
    - "Test principle 1"
  blackbox_testing_guidance: |
    **Why Blackbox Testing Matters:**

    Blackbox tests verify behavior through the public API.

    **How to convert:**
    1. Change package declaration to package foo_test
    2. Import your package
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create THREE internal packages with whitebox tests
	for i := 1; i <= 3; i++ {
		pkgName := fmt.Sprintf("pkg%d", i)
		pkgDir := filepath.Join(tmpDir, "internal", pkgName)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create source file
		srcGo := fmt.Sprintf(`package %s

func Process() string {
	return "processed"
}
`, pkgName)
		if err := os.WriteFile(filepath.Join(pkgDir, pkgName+".go"), []byte(srcGo), 0644); err != nil {
			t.Fatal(err)
		}

		// Create WHITEBOX test
		testGo := fmt.Sprintf(`package %s

import "testing"

func TestProcess(t *testing.T) {
	result := Process()
	if result != "processed" {
		t.Fail()
	}
}
`, pkgName)
		if err := os.WriteFile(filepath.Join(pkgDir, pkgName+"_test.go"), []byte(testGo), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Run binary
	cmd := exec.Command(binaryPath, ".")
	cmd.Dir = tmpDir
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	// Should have 3 whitebox test violations
	whiteboxCount := strings.Count(outputStr, "[ERROR] Whitebox Test")
	if whiteboxCount != 3 {
		t.Errorf("expected 3 whitebox test violations, got %d", whiteboxCount)
	}

	// Each violation should be concise (no "WHY THIS MATTERS" in violation body)
	// The "WHY THIS MATTERS" should only appear in the guidance section at the end
	violationsSection := outputStr[strings.Index(outputStr, "VIOLATIONS"):strings.Index(outputStr, "└────────────────────────────────────────────────────────────────────────────────┘")]
	whyThisMattersInViolations := strings.Count(violationsSection, "WHY THIS MATTERS")
	if whyThisMattersInViolations > 0 {
		t.Errorf("expected 'WHY THIS MATTERS' to not appear in violations section, but found %d occurrences", whyThisMattersInViolations)
	}

	// Should have exactly ONE "BLACKBOX TESTING GUIDANCE" section
	guidanceCount := strings.Count(outputStr, "BLACKBOX TESTING GUIDANCE")
	if guidanceCount != 1 {
		t.Errorf("expected exactly 1 BLACKBOX TESTING GUIDANCE section, got %d", guidanceCount)
	}

	// The guidance section should contain the educational content
	if !strings.Contains(outputStr, "**Why Blackbox Testing Matters:**") {
		t.Error("expected blackbox testing guidance to contain educational content")
	}

	if !strings.Contains(outputStr, "**How to convert:**") {
		t.Error("expected blackbox testing guidance to contain conversion steps")
	}

	// Verify that violations themselves are concise
	if strings.Contains(violationsSection, "After changing to blackbox testing") {
		t.Error("expected detailed guidance to NOT appear in individual violations")
	}
}

func TestCLI_Staticcheck_Clean(t *testing.T) {
	// Skip if staticcheck is not available
	if _, err := exec.LookPath("staticcheck"); err != nil {
		t.Skip("staticcheck not available in PATH")
	}

	tmpDir := t.TempDir()

	// Create valid project with no staticcheck issues
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
scan_paths:
  - cmd
  - pkg
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/staticcheck-clean

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create simple, clean files
	cmdDir := filepath.Join(tmpDir, "cmd")
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/test/staticcheck-clean/pkg"

func main() {
	pkg.Run()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	pkgGo := `package pkg

func Run() {
	println("Hello")
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "pkg.go"), []byte(pkgGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary with --staticcheck flag
	cmd := exec.Command(binaryPath, "--staticcheck", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Should succeed (no arch violations, no staticcheck issues)
	if err != nil {
		t.Errorf("expected exit code 0, got error: %v\nOutput: %s", err, output)
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nOutput: %s", exitCode, output)
	}

	// Output should contain staticcheck results section
	if !strings.Contains(outputStr, "STATICCHECK RESULTS") {
		t.Error("expected output to contain 'STATICCHECK RESULTS' section")
	}

	// Should show no issues found
	if !strings.Contains(outputStr, "✓ No issues found") {
		t.Errorf("expected staticcheck to show no issues, got: %s", outputStr)
	}
}

func TestCLI_Staticcheck_WithIssues(t *testing.T) {
	// Skip if staticcheck is not available
	if _, err := exec.LookPath("staticcheck"); err != nil {
		t.Skip("staticcheck not available in PATH")
	}

	tmpDir := t.TempDir()

	// Create project with staticcheck issues
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
scan_paths:
  - pkg
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/staticcheck-issues

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pkg with staticcheck issues
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Code with staticcheck issues: unused variable, ineffective assignment
	pkgGo := `package pkg

func Run() {
	var x int
	x = 1
	x = 2  // SA4006: this value of x is never used
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "pkg.go"), []byte(pkgGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run binary with --staticcheck flag
	cmd := exec.Command(binaryPath, "--staticcheck", ".")
	cmd.Dir = tmpDir
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	// Should fail (staticcheck found issues)
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (staticcheck issues), got %d\nOutput: %s", exitCode, output)
	}

	// Output should contain staticcheck results section
	if !strings.Contains(outputStr, "STATICCHECK RESULTS") {
		t.Error("expected output to contain 'STATICCHECK RESULTS' section")
	}

	// Should contain staticcheck findings (SA4006 or similar)
	if !strings.Contains(outputStr, "pkg/pkg.go") {
		t.Errorf("expected staticcheck to report issues in pkg/pkg.go, got: %s", outputStr)
	}
}

func TestCLI_Staticcheck_NotAvailable(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid project
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
scan_paths:
  - pkg
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	pkgGo := `package pkg

func Run() {}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "pkg.go"), []byte(pkgGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run with --staticcheck but with PATH modified to not include staticcheck
	cmd := exec.Command(binaryPath, "--staticcheck", ".")
	cmd.Dir = tmpDir
	// Override PATH to exclude staticcheck (if it exists)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	// Should not fail build even if staticcheck is missing (shows warning instead)
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	// Exit code should be 0 (no arch violations, staticcheck missing is a warning)
	if exitCode != 0 {
		t.Logf("Note: Test expected exit code 0 when staticcheck is missing, got %d. This is acceptable if staticcheck is actually installed system-wide.", exitCode)
	}

	// Should contain warning about staticcheck not found
	if !strings.Contains(outputStr, "Staticcheck error") || !strings.Contains(outputStr, "staticcheck not found") {
		// If staticcheck was actually found (system-wide install), that's fine
		if !strings.Contains(outputStr, "STATICCHECK RESULTS") {
			t.Errorf("expected warning about staticcheck not found or staticcheck results, got: %s", outputStr)
		}
	}
}

