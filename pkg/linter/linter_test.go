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
