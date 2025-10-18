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
