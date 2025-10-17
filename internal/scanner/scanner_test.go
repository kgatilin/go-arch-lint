package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/scanner"
)

func TestScan_BasicFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	pkgDir := filepath.Join(tmpDir, "pkg", "service")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	serviceGo := `package service

import (
	"fmt"
	"github.com/test/project/internal/types"
	"github.com/external/lib"
)

func Hello() {
	fmt.Println("hello")
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "service.go"), []byte(serviceGo), 0644); err != nil {
		t.Fatal(err)
	}

	s := scanner.New(tmpDir, "github.com/test/project", nil)
	files, err := s.Scan([]string{"pkg"})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]
	if file.Package != "service" {
		t.Errorf("expected package service, got %s", file.Package)
	}

	if len(file.Imports) != 3 {
		t.Errorf("expected 3 imports, got %d", len(file.Imports))
	}

	expectedImports := map[string]bool{
		"fmt":                                    true,
		"github.com/test/project/internal/types": true,
		"github.com/external/lib":                true,
	}

	for _, imp := range file.Imports {
		if !expectedImports[imp] {
			t.Errorf("unexpected import: %s", imp)
		}
	}
}

func TestScan_IgnoresPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in vendor (should be ignored)
	vendorDir := filepath.Join(tmpDir, "vendor", "lib")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	vendorGo := `package lib
import "fmt"
`
	if err := os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte(vendorGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create normal file
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	normalGo := `package pkg
`
	if err := os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte(normalGo), 0644); err != nil {
		t.Fatal(err)
	}

	s := scanner.New(tmpDir, "github.com/test/project", []string{"vendor"})
	files, err := s.Scan([]string{"pkg", "vendor"})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only find main.go, not lib.go from vendor
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if filepath.Base(files[0].Path) != "main.go" {
		t.Errorf("expected main.go, got %s", filepath.Base(files[0].Path))
	}
}

func TestScan_SkipsTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create normal file
	normalGo := `package pkg
`
	if err := os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte(normalGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test file (should be skipped)
	testGo := `package pkg
import "testing"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "main_test.go"), []byte(testGo), 0644); err != nil {
		t.Fatal(err)
	}

	s := scanner.New(tmpDir, "github.com/test/project", nil)
	files, err := s.Scan([]string{"pkg"})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only find main.go, not main_test.go
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if filepath.Base(files[0].Path) != "main.go" {
		t.Errorf("expected main.go, got %s", filepath.Base(files[0].Path))
	}
}

func TestScan_NonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()

	s := scanner.New(tmpDir, "github.com/test/project", nil)
	files, err := s.Scan([]string{"nonexistent"})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should return empty list, not error
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}
