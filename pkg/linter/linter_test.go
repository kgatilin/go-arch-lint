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
	graphOutput, violationsOutput, err := linter.Run(tmpDir, "markdown")
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
	apiOutput, violationsOutput, err := linter.Run(tmpDir, "api")
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
	_, violationsOutput, err := linter.Run(tmpDir, "markdown")
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
	_, _, err := linter.Run(tmpDir, "markdown")
	if err == nil {
		t.Error("expected error for missing config")
	}
}
