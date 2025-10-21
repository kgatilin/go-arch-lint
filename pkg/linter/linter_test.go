package linter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kgatilin/go-arch-lint/pkg/linter"
)

func TestRun_MarkdownFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
    internal: []
scan_paths:
  - cmd
  - pkg
ignore_paths: []
detect_unused: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd file
	cmdDir := filepath.Join(tmpDir, "cmd", "app")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/test/project/pkg/service"

func main() {
	service.Run()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pkg file
	pkgDir := filepath.Join(tmpDir, "pkg", "service")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	serviceGo := `package service

func Run() {}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "service.go"), []byte(serviceGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter with markdown format
	graphOutput, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check graph output
	if !strings.Contains(graphOutput, "# Dependency Graph") {
		t.Error("missing dependency graph header")
	}

	if !strings.Contains(graphOutput, "cmd/app/main.go") {
		t.Error("missing cmd file in graph")
	}

	if !strings.Contains(graphOutput, "local:pkg/service") {
		t.Error("missing local dependency in graph")
	}

	// No violations expected
	if violationsOutput != "" {
		t.Errorf("expected no violations, got: %s", violationsOutput)
	}
}

func TestRun_APIFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
    internal: []
scan_paths:
  - pkg
ignore_paths: []
detect_unused: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pkg file with exported declarations
	pkgDir := filepath.Join(tmpDir, "pkg", "api")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	apiGo := `package api

type Client struct {
	Name string
}

func NewClient(name string) *Client {
	return &Client{Name: name}
}

const Version = "1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "api.go"), []byte(apiGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter with api format
	apiOutput, violationsOutput, _, err := linter.Run(tmpDir, "api", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check API output
	if !strings.Contains(apiOutput, "# Public API") {
		t.Error("missing Public API header")
	}

	if !strings.Contains(apiOutput, "## api") {
		t.Error("missing api package section")
	}

	if !strings.Contains(apiOutput, "- *Client*") {
		t.Error("missing Client type (should be italic, no methods)")
	}

	if !strings.Contains(apiOutput, "- NewClient") {
		t.Error("missing NewClient function")
	}

	if !strings.Contains(apiOutput, "- Version") {
		t.Error("missing Version constant")
	}

	// No violations for API format
	if violationsOutput != "" {
		t.Errorf("expected no violations for api format, got: %s", violationsOutput)
	}
}

func TestRun_WithViolations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: []
    internal: []
scan_paths:
  - pkg
ignore_paths: []
detect_unused: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pkg files that violate pkg-to-pkg rule
	pkg1Dir := filepath.Join(tmpDir, "pkg", "service1")
	if err := os.MkdirAll(pkg1Dir, 0755); err != nil {
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

	pkg2Dir := filepath.Join(tmpDir, "pkg", "service2")
	if err := os.MkdirAll(pkg2Dir, 0755); err != nil {
		t.Fatal(err)
	}

	service2Go := `package service2

func Helper() {}
`
	if err := os.WriteFile(filepath.Join(pkg2Dir, "service2.go"), []byte(service2Go), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have violations
	if violationsOutput == "" {
		t.Error("expected violations, got none")
	}

	if !strings.Contains(violationsOutput, "DEPENDENCY VIOLATIONS DETECTED") {
		t.Error("missing violations header")
	}

	if !strings.Contains(violationsOutput, "Forbidden pkg-to-pkg Dependency") {
		t.Error("missing violation type")
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// No .goarchlint file
	_, _, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestRun_SharedExternalImports_Detection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config with shared external imports detection
	configYAML := `rules:
  directories_import:
    cmd: [internal]
    internal: []
  detect_unused: false
  shared_external_imports:
    detect: true
    mode: warn
    exclusions:
      - fmt
      - strings
scan_paths:
  - cmd
  - internal
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd file importing external package
	cmdDir := filepath.Join(tmpDir, "cmd")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import (
	"fmt"
	"github.com/pkg/errors"
)

func main() {
	fmt.Println("test")
	err := errors.New("test error")
	_ = err
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create internal file importing same external package
	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	repoGo := `package internal

import "github.com/pkg/errors"

func Query() error {
	return errors.New("query error")
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "repo.go"), []byte(repoGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have violation for github.com/pkg/errors
	if violationsOutput == "" {
		t.Fatal("expected shared external import violation, got none")
	}

	if !strings.Contains(violationsOutput, "Shared External Import") {
		t.Error("missing 'Shared External Import' violation type")
	}

	if !strings.Contains(violationsOutput, "github.com/pkg/errors") {
		t.Error("missing github.com/pkg/errors in violation")
	}

	if !strings.Contains(violationsOutput, "2 layers") {
		t.Error("missing layer count in violation")
	}

	// Should NOT fail in warn mode
	if shouldFail {
		t.Error("expected shouldFail=false in warn mode, got true")
	}

	// Should NOT have violation for fmt (excluded)
	if strings.Contains(violationsOutput, "fmt") {
		t.Error("fmt should be excluded, but was flagged")
	}
}

func TestRun_SharedExternalImports_ExactExclusion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config with github.com/pkg/errors in exclusions
	configYAML := `rules:
  directories_import:
    cmd: [internal]
    internal: []
  detect_unused: false
  shared_external_imports:
    detect: true
    mode: error
    exclusions:
      - github.com/pkg/errors
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

	// Create files importing excluded package
	cmdDir := filepath.Join(tmpDir, "cmd")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
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

	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
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

	// Run linter
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should NOT have violation (excluded)
	if strings.Contains(violationsOutput, "github.com/pkg/errors") {
		t.Error("github.com/pkg/errors should be excluded, but was flagged")
	}
}

func TestRun_SharedExternalImports_GlobPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config with glob pattern
	configYAML := `rules:
  directories_import:
    cmd: [internal]
    internal: []
  detect_unused: false
  shared_external_imports:
    detect: true
    mode: warn
    exclusion_patterns:
      - github.com/pkg/*
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

	// Create files importing package matching glob pattern
	cmdDir := filepath.Join(tmpDir, "cmd")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
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

	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
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

	// Run linter
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should NOT have violation (matches glob pattern)
	if strings.Contains(violationsOutput, "github.com/pkg/errors") {
		t.Error("github.com/pkg/errors should match glob pattern github.com/pkg/*, but was flagged")
	}
}

func TestRun_SharedExternalImports_ErrorMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config with mode: error
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
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
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

	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
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

	// Run linter
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have violation
	if !strings.Contains(violationsOutput, "Shared External Import") {
		t.Error("expected shared external import violation")
	}

	// SHOULD fail in error mode
	if !shouldFail {
		t.Error("expected shouldFail=true in error mode, got false")
	}
}

func TestRun_SharedExternalImports_SingleLayer(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config
	configYAML := `rules:
  directories_import:
    cmd: [internal]
    internal: []
  detect_unused: false
  shared_external_imports:
    detect: true
    mode: error
scan_paths:
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

	// Create multiple files in SAME layer importing same external package
	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	repo1Go := `package internal

import "github.com/pkg/errors"

func Query() error {
	return errors.New("error")
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "repo1.go"), []byte(repo1Go), 0644); err != nil {
		t.Fatal(err)
	}

	repo2Go := `package internal

import "github.com/pkg/errors"

func Save() error {
	return errors.New("save error")
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "repo2.go"), []byte(repo2Go), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should NOT have violation (same layer is OK)
	if strings.Contains(violationsOutput, "Shared External Import") {
		t.Errorf("expected no shared external import violation for same layer, got: %s", violationsOutput)
	}
}

func TestRun_TestFileLinting_Enabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config with test file linting enabled
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: [internal]
    internal: []
  test_files:
    lint: true
    exempt_imports:
      - testing
scan_paths:
  - cmd
  - pkg
  - internal
detect_unused: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd file (production code - clean)
	cmdDir := filepath.Join(tmpDir, "cmd")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/test/project/pkg/service"

func main() {
	service.Run()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd test file that VIOLATES architecture (imports internal directly)
	mainTestGo := `package main

import (
	"testing"
	"github.com/test/project/internal/domain"
)

func TestMain(t *testing.T) {
	domain.DoSomething()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main_test.go"), []byte(mainTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pkg file
	pkgDir := filepath.Join(tmpDir, "pkg", "service")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	serviceGo := `package service

import "github.com/test/project/internal/domain"

func Run() {
	domain.DoSomething()
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "service.go"), []byte(serviceGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create internal file
	internalDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	domainGo := `package domain

func DoSomething() {}
`
	if err := os.WriteFile(filepath.Join(internalDir, "domain.go"), []byte(domainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have a violation for cmd test file importing internal directly
	if !strings.Contains(violationsOutput, "cmd/main_test.go") {
		t.Errorf("expected violation for test file, got: %s", violationsOutput)
	}

	if !strings.Contains(violationsOutput, "internal/domain") {
		t.Errorf("expected violation mentioning internal/domain, got: %s", violationsOutput)
	}

	if !shouldFail {
		t.Error("expected shouldFail=true when test file has violations")
	}
}

func TestRun_TestFileLinting_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config with test file linting disabled (default)
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: [internal]
    internal: []
  test_files:
    lint: false
scan_paths:
  - cmd
  - pkg
  - internal
detect_unused: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd file
	cmdDir := filepath.Join(tmpDir, "cmd")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/test/project/pkg/service"

func main() {
	service.Run()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd test file that would violate architecture (but should be ignored)
	mainTestGo := `package main

import (
	"testing"
	"github.com/test/project/internal/domain"
)

func TestMain(t *testing.T) {
	domain.DoSomething()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main_test.go"), []byte(mainTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pkg file
	pkgDir := filepath.Join(tmpDir, "pkg", "service")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	serviceGo := `package service

import "github.com/test/project/internal/domain"

func Run() {
	domain.DoSomething()
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "service.go"), []byte(serviceGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create internal file
	internalDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	domainGo := `package domain

func DoSomething() {}
`
	if err := os.WriteFile(filepath.Join(internalDir, "domain.go"), []byte(domainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should NOT have a violation for test file (linting disabled)
	if strings.Contains(violationsOutput, "main_test.go") {
		t.Errorf("expected no violation for test file when linting disabled, got: %s", violationsOutput)
	}

	// Production code should be clean
	if shouldFail {
		t.Errorf("expected shouldFail=false, got violations: %s", violationsOutput)
	}
}

// TestRun_BlackBoxTests tests that black-box tests can import their parent package
func TestRun_BlackBoxTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config with test file linting enabled
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: [internal]
    internal: []
  detect_unused: false
  test_files:
    lint: true
scan_paths:
  - cmd
  - pkg
  - internal
ignore_paths: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create internal package
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

	// Create BLACK-BOX test file (package app_test) importing parent package
	// This should NOT create a violation
	blackBoxTestGo := `package app_test

import (
	"testing"
	"github.com/test/project/internal/app"
)

func TestProcess(t *testing.T) {
	result := app.Process()
	if result != "processed" {
		t.Errorf("expected 'processed', got %s", result)
	}
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "app_blackbox_test.go"), []byte(blackBoxTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create another internal package for testing white-box tests
	configDir := filepath.Join(tmpDir, "internal", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configGo := `package config

func Load() string {
	return "loaded"
}
`
	if err := os.WriteFile(filepath.Join(configDir, "config.go"), []byte(configGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create WHITE-BOX test file (package app) importing another internal package
	// This SHOULD create a violation (internal packages can't import each other)
	whiteBoxTestGo := `package app

import (
	"testing"
	"github.com/test/project/internal/config"
)

func TestWithConfig(t *testing.T) {
	_ = config.Load()
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "app_whitebox_test.go"), []byte(whiteBoxTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd file to avoid empty cmd directory issues
	cmdDir := filepath.Join(tmpDir, "cmd", "main")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/test/project/internal/app"

func main() {
	app.Process()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have violation for white-box test importing another internal package
	if !shouldFail {
		t.Error("expected shouldFail=true due to white-box test violation")
	}

	// Should have violation for white-box test
	if !strings.Contains(violationsOutput, "app_whitebox_test.go") {
		t.Error("expected violation for white-box test importing another internal package")
	}

	// Should NOT have violation for black-box test
	if strings.Contains(violationsOutput, "app_blackbox_test.go") {
		t.Errorf("did not expect violation for black-box test, but got: %s", violationsOutput)
	}
}

// TestRun_BlackBoxTests_MultipleImports tests that black-box tests' other imports follow normal architecture rules
func TestRun_BlackBoxTests_MultipleImports(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config with test file linting enabled
	configYAML := `rules:
  directories_import:
    cmd: [pkg]
    pkg: [internal]
    internal: []
  detect_unused: false
  test_files:
    lint: true
scan_paths:
  - cmd
  - pkg
  - internal
ignore_paths: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create internal packages
	appDir := filepath.Join(tmpDir, "internal", "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	appGo := `package app

func Process() string {
	return "processed"
}
`
	if err := os.WriteFile(filepath.Join(appDir, "app.go"), []byte(appGo), 0644); err != nil {
		t.Fatal(err)
	}

	configDir := filepath.Join(tmpDir, "internal", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configGo := `package config

func Load() string {
	return "loaded"
}
`
	if err := os.WriteFile(filepath.Join(configDir, "config.go"), []byte(configGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create BLACK-BOX test importing BOTH parent (exempted) and another internal package (violates normal rules)
	blackBoxTestGo := `package app_test

import (
	"testing"
	"github.com/test/project/internal/app"
	"github.com/test/project/internal/config"
)

func TestProcessWithConfig(t *testing.T) {
	_ = app.Process()
	_ = config.Load()
}
`
	if err := os.WriteFile(filepath.Join(appDir, "app_test.go"), []byte(blackBoxTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd file
	cmdDir := filepath.Join(tmpDir, "cmd", "main")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "github.com/test/project/internal/app"

func main() {
	app.Process()
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have violation for importing internal/config (violates internal:[] architecture rule)
	if !shouldFail {
		t.Error("expected shouldFail=true due to forbidden import violation")
	}

	// Should have violation mentioning internal/config
	if !strings.Contains(violationsOutput, "internal/config") {
		t.Error("expected violation for importing internal/config (violates internal:[] rule)")
	}

	// Check individual violations
	violations := strings.Split(violationsOutput, "[ERROR]")
	foundAppTestViolatingParent := false
	foundAppTestViolatingConfig := false

	for _, viol := range violations {
		if strings.Contains(viol, "app_test.go") {
			if strings.Contains(viol, "imports internal/app") {
				foundAppTestViolatingParent = true
			}
			if strings.Contains(viol, "imports internal/config") {
				foundAppTestViolatingConfig = true
			}
		}
	}

	if foundAppTestViolatingParent {
		t.Error("did not expect violation for app_test.go importing parent package internal/app")
	}

	if !foundAppTestViolatingConfig {
		t.Error("expected violation for app_test.go importing internal/config")
	}
}

// TestRun_RequireBlackboxTests tests that whitebox tests are detected when rule is enabled
func TestRun_RequireBlackboxTests(t *testing.T) {
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

	// Create another package with BLACKBOX test (should NOT violate)
	configDir := filepath.Join(tmpDir, "internal", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configGo := `package config

func Load() string {
	return "loaded"
}
`
	if err := os.WriteFile(filepath.Join(configDir, "config.go"), []byte(configGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create BLACKBOX test (correct)
	blackboxTestGo := `package config_test

import (
	"testing"
	"github.com/test/project/internal/config"
)

func TestLoad(t *testing.T) {
	result := config.Load()
	if result != "loaded" {
		t.Fail()
	}
}
`
	if err := os.WriteFile(filepath.Join(configDir, "config_test.go"), []byte(blackboxTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should fail due to whitebox test violation
	if !shouldFail {
		t.Error("expected shouldFail=true due to whitebox test violation")
	}

	// Should contain whitebox test violation
	if !strings.Contains(violationsOutput, "Whitebox Test") {
		t.Error("expected output to contain 'Whitebox Test' violation")
	}

	// Should mention the whitebox test file
	if !strings.Contains(violationsOutput, "internal/app/app_test.go") {
		t.Error("expected violation for internal/app/app_test.go")
	}

	// Should mention the expected package name
	if !strings.Contains(violationsOutput, "app_test") {
		t.Error("expected violation to suggest package name 'app_test'")
	}

	// Should NOT have violation for blackbox test
	if strings.Contains(violationsOutput, "internal/config/config_test.go") {
		t.Error("did not expect violation for blackbox test internal/config/config_test.go")
	}

	// Violations should be concise (no error_prompt enabled in this test)
	if !strings.Contains(violationsOutput, "Blackbox testing is enforced") {
		t.Error("expected violation to contain rule about blackbox testing")
	}

	if !strings.Contains(violationsOutput, "Change package declaration from 'package app' to 'package app_test'") {
		t.Error("expected violation to contain fix instruction")
	}
}

// TestPresets_RequireBlackboxDefault verifies all presets have require_blackbox enabled by default
func TestAvailablePresets(t *testing.T) {
	presets := linter.AvailablePresets()

	if len(presets) == 0 {
		t.Fatal("expected at least one preset")
	}

	// Check that ddd preset exists
	foundDDD := false
	for _, p := range presets {
		if p.Name == "ddd" {
			foundDDD = true
			if p.Description == "" {
				t.Error("ddd preset missing description")
			}
			if p.ArchitecturalGoals == "" {
				t.Error("ddd preset missing architectural goals")
			}
			if len(p.Principles) == 0 {
				t.Error("ddd preset missing principles")
			}
		}
	}

	if !foundDDD {
		t.Error("expected ddd preset to be available")
	}
}

func TestGetPreset_ValidPreset(t *testing.T) {
	preset, err := linter.GetPreset("ddd")
	if err != nil {
		t.Fatalf("GetPreset('ddd') failed: %v", err)
	}

	if preset.Name != "ddd" {
		t.Errorf("expected preset name 'ddd', got '%s'", preset.Name)
	}

	if preset.Description == "" {
		t.Error("preset missing description")
	}

	if len(preset.Config.Structure.RequiredDirectories) == 0 {
		t.Error("preset missing required directories")
	}

	if len(preset.Config.Rules.DirectoriesImport) == 0 {
		t.Error("preset missing import rules")
	}
}

func TestGetPreset_InvalidPreset(t *testing.T) {
	_, err := linter.GetPreset("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent preset")
	}
}

func TestInit_WithDDDPreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize with ddd preset
	err := linter.Init(tmpDir, "ddd", true)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check .goarchlint was created
	configPath := filepath.Join(tmpDir, ".goarchlint")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal(".goarchlint file was not created")
	}

	// Read and verify config
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// Check for required directories from ddd preset
	if !strings.Contains(content, "internal/domain") {
		t.Error("config missing internal/domain directory")
	}
	if !strings.Contains(content, "internal/app") {
		t.Error("config missing internal/app directory")
	}
	if !strings.Contains(content, "internal/infra") {
		t.Error("config missing internal/infra directory")
	}

	// Check for error_prompt section
	if !strings.Contains(content, "error_prompt:") {
		t.Error("config missing error_prompt section")
	}
	if !strings.Contains(content, "architectural_goals:") {
		t.Error("config missing architectural_goals")
	}

	// Check that required directories were created (createDirs=true)
	for _, dir := range []string{"internal/domain", "internal/app", "internal/infra", "cmd"} {
		dirPath := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("directory %s was not created", dir)
		}
	}
}

func TestInit_WithSimplePreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize with simple preset
	err := linter.Init(tmpDir, "simple", false)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check .goarchlint was created
	configPath := filepath.Join(tmpDir, ".goarchlint")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal(".goarchlint file was not created")
	}

	// With createDirs=false, directories should NOT be created
	cmdPath := filepath.Join(tmpDir, "cmd")
	if _, err := os.Stat(cmdPath); err == nil {
		t.Error("directories should not be created when createDirs=false")
	}
}

func TestInit_InvalidPreset(t *testing.T) {
	tmpDir := t.TempDir()

	err := linter.Init(tmpDir, "invalid-preset", false)
	if err == nil {
		t.Error("expected error for invalid preset")
	}
}

func TestInit_ConfigAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create existing config
	existingConfig := `rules:
  directories_import:
    cmd: [pkg]
`
	configPath := filepath.Join(tmpDir, ".goarchlint")
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to init - should fail
	err := linter.Init(tmpDir, "ddd", false)
	if err == nil {
		t.Error("expected error when .goarchlint already exists")
	}
}

func TestRefresh_WithSamePreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// First initialize with ddd preset
	if err := linter.Init(tmpDir, "ddd", false); err != nil {
		t.Fatal(err)
	}

	// Modify the config to add custom rule
	configPath := filepath.Join(tmpDir, ".goarchlint")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Add a comment to verify it's preserved
	modifiedConfig := "# Custom comment\n" + string(data)
	if err := os.WriteFile(configPath, []byte(modifiedConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Refresh with same preset (empty string means use current)
	if err := linter.Refresh(tmpDir, ""); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	// Read refreshed config
	refreshedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	refreshedContent := string(refreshedData)

	// Should still have error_prompt section
	if !strings.Contains(refreshedContent, "error_prompt:") {
		t.Error("refreshed config missing error_prompt section")
	}

	// Should still have architectural_goals
	if !strings.Contains(refreshedContent, "architectural_goals:") {
		t.Error("refreshed config missing architectural_goals")
	}
}

func TestRefresh_SwitchToNewPreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize with simple preset
	if err := linter.Init(tmpDir, "simple", false); err != nil {
		t.Fatal(err)
	}

	// Refresh with ddd preset (switch)
	if err := linter.Refresh(tmpDir, "ddd"); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	// Read config
	configPath := filepath.Join(tmpDir, ".goarchlint")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// Should now have DDD-specific content (new format with preset section)
	if !strings.Contains(content, "preset:") && !strings.Contains(content, "preset_used: ddd") {
		t.Error("config should indicate ddd preset")
	}
	if !strings.Contains(content, "name: ddd") && !strings.Contains(content, "preset_used: ddd") {
		t.Error("config should contain ddd preset name")
	}

	// Should have error_prompt from ddd
	if !strings.Contains(content, "error_prompt:") {
		t.Error("config missing error_prompt after switching preset")
	}
}

func TestRefresh_NoConfigExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to refresh without existing config - should fail
	err := linter.Refresh(tmpDir, "ddd")
	if err == nil {
		t.Error("expected error when refreshing without existing config")
	}
}

func TestPresets_RequireBlackboxDefault(t *testing.T) {
	presets := linter.AvailablePresets()

	if len(presets) == 0 {
		t.Fatal("expected at least one preset")
	}

	for _, preset := range presets {
		if !preset.Config.Rules.TestFiles.RequireBlackbox {
			t.Errorf("preset '%s' should have require_blackbox: true by default, but it's false", preset.Name)
		}

		// Also verify that test linting is enabled
		if !preset.Config.Rules.TestFiles.Lint {
			t.Errorf("preset '%s' should have test_files.lint: true, but it's false", preset.Name)
		}
	}
}

func TestRun_StrictTestNaming_MissingTestFile_NoViolation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config with strict_test_naming enabled
	configYAML := `rules:
  directories_import:
    pkg: []
  strict_test_naming: true
  test_files:
    lint: true
scan_paths:
  - pkg
ignore_paths: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pkg file WITHOUT test file
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	fooGo := `package pkg

func Foo() string {
	return "foo"
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "foo.go"), []byte(fooGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have NO violations - missing test files are handled by coverage metrics
	if violationsOutput != "" {
		t.Errorf("expected no violations for missing test file (coverage handles this), got: %s", violationsOutput)
	}
}

func TestRun_StrictTestNaming_OrphanedTestFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config with strict_test_naming enabled
	configYAML := `rules:
  directories_import:
    pkg: []
  strict_test_naming: true
  test_files:
    lint: true
scan_paths:
  - pkg
ignore_paths: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test file WITHOUT implementation file
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	fooTestGo := `package pkg_test

import "testing"

func TestFoo(t *testing.T) {
	// Test without implementation
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "foo_test.go"), []byte(fooTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have violation for orphaned test file
	if violationsOutput == "" {
		t.Fatal("expected violations for orphaned test file, got none")
	}

	if !strings.Contains(violationsOutput, "Test Naming Convention") {
		t.Errorf("expected 'Test Naming Convention' violation, got: %s", violationsOutput)
	}

	if !strings.Contains(violationsOutput, "no corresponding implementation file") {
		t.Errorf("expected error about orphaned test file, got: %s", violationsOutput)
	}

	if !strings.Contains(violationsOutput, "foo_test.go") {
		t.Errorf("expected violation to reference foo_test.go, got: %s", violationsOutput)
	}
}

func TestRun_StrictTestNaming_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config with strict_test_naming enabled
	configYAML := `rules:
  directories_import:
    pkg: []
  strict_test_naming: true
  test_files:
    lint: true
scan_paths:
  - pkg
ignore_paths: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create implementation and test file (valid 1:1)
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	fooGo := `package pkg

func Foo() string {
	return "foo"
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "foo.go"), []byte(fooGo), 0644); err != nil {
		t.Fatal(err)
	}

	fooTestGo := `package pkg_test

import "testing"

func TestFoo(t *testing.T) {
	// Valid test
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "foo_test.go"), []byte(fooTestGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have no violations
	if violationsOutput != "" {
		t.Errorf("expected no violations for valid 1:1 mapping, got: %s", violationsOutput)
	}
}

func TestRun_StrictTestNaming_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .goarchlint config WITHOUT strict_test_naming
	configYAML := `rules:
  directories_import:
    pkg: []
  test_files:
    lint: true
scan_paths:
  - pkg
ignore_paths: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Create implementation WITHOUT test file
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	fooGo := `package pkg

func Foo() string {
	return "foo"
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "foo.go"), []byte(fooGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run linter
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false, false, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have no violations when feature is disabled
	if violationsOutput != "" {
		t.Errorf("expected no violations when strict_test_naming is disabled, got: %s", violationsOutput)
	}
}
