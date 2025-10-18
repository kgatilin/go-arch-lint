package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary builds the go-arch-lint binary for testing
func buildBinary(t *testing.T) string {
	t.Helper()

	// Build in temp directory
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "go-arch-lint")

	// Get project root (two levels up from cmd/go-arch-lint)
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/go-arch-lint")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\nOutput: %s", err, output)
	}

	return binaryPath
}

func TestCLI_NoViolations_ExitCode0(t *testing.T) {
	binary := buildBinary(t)
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
	cmd := exec.Command(binary, ".")
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
	binary := buildBinary(t)
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
	cmd := exec.Command(binary, ".")
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
	binary := buildBinary(t)
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
	cmd := exec.Command(binary, "-exit-zero", ".")
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
	binary := buildBinary(t)
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
	cmd := exec.Command(binary, ".")
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
	binary := buildBinary(t)
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
	cmd := exec.Command(binary, ".")
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
	binary := buildBinary(t)
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
	cmd := exec.Command(binary, "-format=markdown", ".")
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
	binary := buildBinary(t)
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Run init with ddd preset
	cmd := exec.Command(binary, "init", "--preset=ddd")
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
	binary := buildBinary(t)
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Run init with simple preset
	cmd := exec.Command(binary, "init", "--preset=simple")
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
	binary := buildBinary(t)
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Run init with hexagonal preset
	cmd := exec.Command(binary, "init", "--preset=hexagonal")
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
