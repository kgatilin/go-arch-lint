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

	s := scanner.New(tmpDir, "github.com/test/project", nil, false)
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

	s := scanner.New(tmpDir, "github.com/test/project", []string{"vendor"}, false)
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

	s := scanner.New(tmpDir, "github.com/test/project", nil, false)
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

	s := scanner.New(tmpDir, "github.com/test/project", nil, false)
	files, err := s.Scan([]string{"nonexistent"})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should return empty list, not error
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestScanWithAPI_ExtractsExportedDeclarations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with various exported declarations
	pkgDir := filepath.Join(tmpDir, "pkg", "api")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	apiGo := `package api

import "context"

// Exported type
type Service struct {
	Name string
}

// Exported function
func NewService(name string) *Service {
	return &Service{Name: name}
}

// Exported method
func (s *Service) GetName() string {
	return s.Name
}

// Exported method with multiple params and returns
func (s *Service) Process(ctx context.Context, data []byte) (string, error) {
	return "", nil
}

// unexported function
func helper() {}

// Exported constant
const MaxRetries = 3

// Exported variable
var DefaultTimeout = 30

// unexported var
var internal = 10
`
	if err := os.WriteFile(filepath.Join(pkgDir, "api.go"), []byte(apiGo), 0644); err != nil {
		t.Fatal(err)
	}

	s := scanner.New(tmpDir, "github.com/test/project", nil, false)
	files, err := s.ScanWithAPI([]string{"pkg"})
	if err != nil {
		t.Fatalf("ScanWithAPI failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]
	if file.Package != "api" {
		t.Errorf("expected package api, got %s", file.Package)
	}

	// Check exported declarations
	decls := file.ExportedDecls
	if len(decls) == 0 {
		t.Fatal("expected exported declarations, got none")
	}

	// Verify we have the expected types
	hasService := false
	hasNewService := false
	hasGetName := false
	hasProcess := false
	hasMaxRetries := false
	hasDefaultTimeout := false

	for _, decl := range decls {
		switch {
		case decl.Name == "Service" && decl.Kind == "type":
			hasService = true
		case decl.Name == "NewService" && decl.Kind == "func":
			hasNewService = true
		case decl.Name == "GetName" && decl.Kind == "func":
			hasGetName = true
		case decl.Name == "Process" && decl.Kind == "func":
			hasProcess = true
		case decl.Name == "MaxRetries" && decl.Kind == "const":
			hasMaxRetries = true
		case decl.Name == "DefaultTimeout" && decl.Kind == "var":
			hasDefaultTimeout = true
		case decl.Name == "helper" || decl.Name == "internal":
			t.Errorf("unexported declaration should not be included: %s", decl.Name)
		}
	}

	if !hasService {
		t.Error("missing Service type")
	}
	if !hasNewService {
		t.Error("missing NewService function")
	}
	if !hasGetName {
		t.Error("missing GetName method")
	}
	if !hasProcess {
		t.Error("missing Process method")
	}
	if !hasMaxRetries {
		t.Error("missing MaxRetries constant")
	}
	if !hasDefaultTimeout {
		t.Error("missing DefaultTimeout variable")
	}
}

func TestScanWithAPI_InterfaceMethods(t *testing.T) {
	tmpDir := t.TempDir()

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	testGo := `package pkg

type User struct {
	Name string
}

const Version = "1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "test.go"), []byte(testGo), 0644); err != nil {
		t.Fatal(err)
	}

	s := scanner.New(tmpDir, "github.com/test/project", nil, false)
	files, err := s.ScanWithAPI([]string{"pkg"})
	if err != nil {
		t.Fatalf("ScanWithAPI failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]

	// Test interface methods
	if file.GetPackage() != "pkg" {
		t.Errorf("GetPackage() = %s, want pkg", file.GetPackage())
	}

	if file.GetRelPath() == "" {
		t.Error("GetRelPath() should not be empty")
	}

	// Test ExportedDecl interface methods
	if len(file.ExportedDecls) > 0 {
		decl := file.ExportedDecls[0]
		if decl.GetName() == "" {
			t.Error("GetName() should not be empty")
		}
		if decl.GetKind() == "" {
			t.Error("GetKind() should not be empty")
		}
		if decl.GetSignature() == "" {
			t.Error("GetSignature() should not be empty")
		}
	}
}

func TestScanWithAPI_ComplexSignatures(t *testing.T) {
	tmpDir := t.TempDir()

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	complexGo := `package pkg

import "context"

// Function with variadic params
func Log(format string, args ...interface{}) {}

// Function with map and slice types
func Process(data map[string][]byte) error {
	return nil
}

// Function with channel
func Watch(ctx context.Context) chan string {
	return nil
}

// Function with function type
func Apply(fn func(string) string) {}

// Method with pointer receiver
func (*Handler) Handle() {}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "complex.go"), []byte(complexGo), 0644); err != nil {
		t.Fatal(err)
	}

	s := scanner.New(tmpDir, "github.com/test/project", nil, false)
	files, err := s.ScanWithAPI([]string{"pkg"})
	if err != nil {
		t.Fatalf("ScanWithAPI failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]
	if len(file.ExportedDecls) == 0 {
		t.Fatal("expected exported declarations")
	}

	// Verify we can build signatures for complex types
	for _, decl := range file.ExportedDecls {
		if decl.Signature == "" {
			t.Errorf("empty signature for %s", decl.Name)
		}
		if decl.Signature == "unknown" {
			t.Errorf("unknown signature for %s", decl.Name)
		}
	}
}

func TestScan_LintTestFiles_Enabled(t *testing.T) {
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

	// Create test file
	testGo := `package pkg
import "testing"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "main_test.go"), []byte(testGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Scan with lintTestFiles=true
	s := scanner.New(tmpDir, "github.com/test/project", nil, true)
	files, err := s.Scan([]string{"pkg"})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find BOTH main.go and main_test.go
	if len(files) != 2 {
		t.Fatalf("expected 2 files (including test file), got %d", len(files))
	}

	// Verify we have both files
	foundMain := false
	foundTest := false
	for _, file := range files {
		base := filepath.Base(file.Path)
		if base == "main.go" {
			foundMain = true
		}
		if base == "main_test.go" {
			foundTest = true
		}
	}

	if !foundMain {
		t.Error("expected to find main.go")
	}
	if !foundTest {
		t.Error("expected to find main_test.go when lintTestFiles=true")
	}
}

func TestScan_LintTestFiles_Disabled(t *testing.T) {
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

	// Create test file
	testGo := `package pkg
import "testing"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "main_test.go"), []byte(testGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Scan with lintTestFiles=false
	s := scanner.New(tmpDir, "github.com/test/project", nil, false)
	files, err := s.Scan([]string{"pkg"})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find ONLY main.go, not main_test.go
	if len(files) != 1 {
		t.Fatalf("expected 1 file (excluding test file), got %d", len(files))
	}

	if filepath.Base(files[0].Path) != "main.go" {
		t.Errorf("expected main.go, got %s", filepath.Base(files[0].Path))
	}
}
