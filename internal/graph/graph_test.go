package graph_test

import (
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/graph"
)

// testFileInfo implements graph.FileInfo interface for testing
type testFileInfo struct {
	relPath string
	pkg     string
	imports []string
}

func (t testFileInfo) GetRelPath() string  { return t.relPath }
func (t testFileInfo) GetPackage() string  { return t.pkg }
func (t testFileInfo) GetImports() []string { return t.imports }

func TestBuild_LocalAndExternalImports(t *testing.T) {
	files := []graph.FileInfo{
		testFileInfo{
			relPath: "pkg/service/service.go",
			pkg:     "service",
			imports: []string{
				"fmt",
				"github.com/test/project/internal/types",
				"github.com/external/lib",
			},
		},
	}

	g := graph.Build(files, "github.com/test/project")

	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}

	node := g.Nodes[0]
	if len(node.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(node.Dependencies))
	}

	// Check local import is classified correctly
	var foundLocal bool
	for _, dep := range node.Dependencies {
		if dep.ImportPath == "github.com/test/project/internal/types" {
			foundLocal = true
			if !dep.IsLocal {
				t.Error("expected internal/types to be local")
			}
			if dep.LocalPath != "internal/types" {
				t.Errorf("expected LocalPath internal/types, got %s", dep.LocalPath)
			}
		}
	}

	if !foundLocal {
		t.Error("did not find local import")
	}

	// Check external import
	var foundExternal bool
	for _, dep := range node.Dependencies {
		if dep.ImportPath == "github.com/external/lib" {
			foundExternal = true
			if dep.IsLocal {
				t.Error("expected external/lib to be external")
			}
		}
	}

	if !foundExternal {
		t.Error("did not find external import")
	}
}

func TestIsStdLib(t *testing.T) {
	tests := []struct {
		importPath string
		want       bool
	}{
		{"fmt", true},
		{"os", true},
		{"path/filepath", true},
		{"github.com/user/repo", false},
		{"gopkg.in/yaml.v3", false},
		{"google.golang.org/grpc", false},
	}

	for _, tt := range tests {
		got := graph.IsStdLib(tt.importPath)
		if got != tt.want {
			t.Errorf("IsStdLib(%q) = %v, want %v", tt.importPath, got, tt.want)
		}
	}
}

func TestGetLocalPackages(t *testing.T) {
	files := []graph.FileInfo{
		testFileInfo{
			relPath: "pkg/service/service.go",
			pkg:     "service",
		},
		testFileInfo{
			relPath: "pkg/service/handler.go",
			pkg:     "service",
		},
		testFileInfo{
			relPath: "internal/types/types.go",
			pkg:     "types",
		},
	}

	g := graph.Build(files, "github.com/test/project")
	packages := g.GetLocalPackages()

	if len(packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(packages))
	}

	pkgMap := make(map[string]bool)
	for _, pkg := range packages {
		pkgMap[pkg] = true
	}

	if !pkgMap["pkg/service"] {
		t.Error("expected pkg/service in packages")
	}

	if !pkgMap["internal/types"] {
		t.Error("expected internal/types in packages")
	}
}

// TestDependency_InterfaceMethods tests Dependency interface methods
func TestDependency_InterfaceMethods(t *testing.T) {
	files := []graph.FileInfo{
		testFileInfo{
			relPath: "pkg/service/service.go",
			pkg:     "service",
			imports: []string{
				"fmt",
				"github.com/test/project/internal/types",
			},
		},
	}

	g := graph.Build(files, "github.com/test/project")

	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}

	node := g.Nodes[0]
	if len(node.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(node.Dependencies))
	}

	// Test local dependency
	var localDep *graph.Dependency
	for i := range node.Dependencies {
		if node.Dependencies[i].IsLocal {
			localDep = &node.Dependencies[i]
			break
		}
	}

	if localDep == nil {
		t.Fatal("expected to find local dependency")
	}

	if localDep.GetImportPath() != "github.com/test/project/internal/types" {
		t.Errorf("GetImportPath() = %s, want github.com/test/project/internal/types", localDep.GetImportPath())
	}

	if localDep.GetLocalPath() != "internal/types" {
		t.Errorf("GetLocalPath() = %s, want internal/types", localDep.GetLocalPath())
	}

	if !localDep.IsLocalDep() {
		t.Error("IsLocalDep() = false, want true")
	}

	// Test external dependency
	var externalDep *graph.Dependency
	for i := range node.Dependencies {
		if !node.Dependencies[i].IsLocal {
			externalDep = &node.Dependencies[i]
			break
		}
	}

	if externalDep == nil {
		t.Fatal("expected to find external dependency")
	}

	if externalDep.GetImportPath() != "fmt" {
		t.Errorf("GetImportPath() = %s, want fmt", externalDep.GetImportPath())
	}

	if externalDep.IsLocalDep() {
		t.Error("IsLocalDep() = true, want false for stdlib")
	}
}

// TestFileNode_InterfaceMethods tests FileNode interface methods
func TestFileNode_InterfaceMethods(t *testing.T) {
	files := []graph.FileInfo{
		testFileInfo{
			relPath: "pkg/service/service.go",
			pkg:     "service",
			imports: []string{"fmt"},
		},
	}

	g := graph.Build(files, "github.com/test/project")

	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}

	node := g.Nodes[0]

	if node.GetRelPath() != "pkg/service/service.go" {
		t.Errorf("GetRelPath() = %s, want pkg/service/service.go", node.GetRelPath())
	}

	if node.GetPackage() != "service" {
		t.Errorf("GetPackage() = %s, want service", node.GetPackage())
	}
}

// TestBuild_MultipleFiles tests Build with multiple files
func TestBuild_MultipleFiles(t *testing.T) {
	files := []graph.FileInfo{
		testFileInfo{
			relPath: "pkg/http/server.go",
			pkg:     "http",
			imports: []string{
				"fmt",
				"github.com/test/project/pkg/database",
			},
		},
		testFileInfo{
			relPath: "pkg/database/db.go",
			pkg:     "database",
			imports: []string{
				"database/sql",
			},
		},
	}

	g := graph.Build(files, "github.com/test/project")

	if len(g.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.Nodes))
	}

	// Verify both nodes exist
	var httpNode, dbNode *graph.FileNode
	for i := range g.Nodes {
		if g.Nodes[i].Package == "http" {
			httpNode = &g.Nodes[i]
		}
		if g.Nodes[i].Package == "database" {
			dbNode = &g.Nodes[i]
		}
	}

	if httpNode == nil {
		t.Fatal("expected to find http node")
	}
	if dbNode == nil {
		t.Fatal("expected to find database node")
	}

	// Verify dependencies
	if len(httpNode.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies for http node, got %d", len(httpNode.Dependencies))
	}

	if len(dbNode.Dependencies) != 1 {
		t.Errorf("expected 1 dependency for database node, got %d", len(dbNode.Dependencies))
	}
}

// TestBuildDetailed tests BuildDetailed with symbol usage tracking
func TestBuildDetailed(t *testing.T) {
	files := []graph.FileInfo{
		testFileInfo{
			relPath: "pkg/service/service.go",
			pkg:     "service",
			imports: []string{
				"fmt",
				"github.com/test/project/internal/types",
			},
		},
	}

	// Usage map: file path -> (import path -> used symbols)
	usageMap := map[string]map[string][]string{
		"pkg/service/service.go": {
			"fmt":                                    {"Println", "Printf"},
			"github.com/test/project/internal/types": {"User", "NewUser"},
		},
	}

	g := graph.BuildDetailed(files, "github.com/test/project", usageMap)

	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}

	node := g.Nodes[0]
	if len(node.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(node.Dependencies))
	}

	// Check fmt dependency has used symbols
	fmtDep := findDependency(node.Dependencies, "fmt")
	if fmtDep == nil {
		t.Fatal("expected to find fmt dependency")
	}

	symbols := fmtDep.GetUsedSymbols()
	if len(symbols) != 2 {
		t.Errorf("expected 2 used symbols from fmt, got %d", len(symbols))
	}

	// Check internal/types dependency
	typesDep := findDependency(node.Dependencies, "github.com/test/project/internal/types")
	if typesDep == nil {
		t.Fatal("expected to find internal/types dependency")
	}

	if !typesDep.IsLocal {
		t.Error("expected internal/types to be local")
	}

	typesSymbols := typesDep.GetUsedSymbols()
	if len(typesSymbols) != 2 {
		t.Errorf("expected 2 used symbols from internal/types, got %d", len(typesSymbols))
	}
}

// TestBuildDetailed_WithNilUsageMap tests BuildDetailed with nil usage map
func TestBuildDetailed_WithNilUsageMap(t *testing.T) {
	files := []graph.FileInfo{
		testFileInfo{
			relPath: "pkg/service/service.go",
			pkg:     "service",
			imports: []string{"fmt"},
		},
	}

	// Nil usage map should still work
	g := graph.BuildDetailed(files, "github.com/test/project", nil)

	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}

	node := g.Nodes[0]
	if len(node.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(node.Dependencies))
	}

	// Should still create dependency, just without used symbols
	fmtDep := &node.Dependencies[0]
	if fmtDep.ImportPath != "fmt" {
		t.Errorf("expected fmt dependency, got %s", fmtDep.ImportPath)
	}

	// Used symbols should be empty or nil
	symbols := fmtDep.GetUsedSymbols()
	if len(symbols) != 0 {
		t.Errorf("expected 0 used symbols with nil usage map, got %d", len(symbols))
	}
}

// TestIsStdLib_EdgeCases tests additional stdlib detection cases
func TestIsStdLib_EdgeCases(t *testing.T) {
	tests := []struct {
		importPath string
		want       bool
	}{
		{"context", true},
		{"encoding/json", true},
		{"net/http", true},
		{"golang.org/x/tools", false}, // x/ packages are not stdlib
	}

	for _, tt := range tests {
		got := graph.IsStdLib(tt.importPath)
		if got != tt.want {
			t.Errorf("IsStdLib(%q) = %v, want %v", tt.importPath, got, tt.want)
		}
	}
}

// Helper function to find dependency by import path
func findDependency(deps []graph.Dependency, importPath string) *graph.Dependency {
	for i := range deps {
		if deps[i].ImportPath == importPath {
			return &deps[i]
		}
	}
	return nil
}
