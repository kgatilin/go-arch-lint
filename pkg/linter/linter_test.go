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
	graphOutput, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false)
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
	apiOutput, violationsOutput, _, err := linter.Run(tmpDir, "api", false)
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
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false)
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
	_, _, _, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, _, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false)
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
	_, violationsOutput, shouldFail, err := linter.Run(tmpDir, "markdown", false)
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

	// Should contain educational prompt about WHY blackbox testing matters
	expectedEducationalContent := []string{
		"WHY THIS MATTERS",
		"public API",
		"internal implementation",
		"Go best practice",
	}

	for _, content := range expectedEducationalContent {
		if !strings.Contains(violationsOutput, content) {
			t.Errorf("expected violation output to contain educational content: '%s'", content)
		}
	}
}

// TestPresets_RequireBlackboxDefault verifies all presets have require_blackbox enabled by default
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
