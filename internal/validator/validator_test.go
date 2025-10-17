package validator

import (
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/config"
	"github.com/kgatilin/go-arch-lint/internal/graph"
	"github.com/kgatilin/go-arch-lint/internal/scanner"
)

// Helper to convert []scanner.FileInfo to []graph.FileInfo (slice covariance workaround)
func toGraphFiles(files []scanner.FileInfo) []graph.FileInfo {
	result := make([]graph.FileInfo, len(files))
	for i := range files {
		result[i] = files[i]
	}
	return result
}

// Test adapter to convert graph.Graph to validator.Graph interface
type testGraphAdapter struct {
	g *graph.Graph
}

func (tga *testGraphAdapter) GetNodes() []FileNode {
	nodes := make([]FileNode, len(tga.g.Nodes))
	for i := range tga.g.Nodes {
		nodes[i] = &testFileNodeAdapter{node: &tga.g.Nodes[i]}
	}
	return nodes
}

type testFileNodeAdapter struct {
	node *graph.FileNode
}

func (tfna *testFileNodeAdapter) GetRelPath() string {
	return tfna.node.RelPath
}

func (tfna *testFileNodeAdapter) GetDependencies() []Dependency {
	deps := make([]Dependency, len(tfna.node.Dependencies))
	for i := range tfna.node.Dependencies {
		deps[i] = &tfna.node.Dependencies[i]
	}
	return deps
}

func TestValidate_PkgToPkgViolation(t *testing.T) {
	files := []scanner.FileInfo{
		{
			RelPath: "pkg/http/server.go",
			Package: "http",
			Imports: []string{
				"github.com/test/project/pkg/database",
			},
		},
		{
			RelPath: "pkg/database/db.go",
			Package: "database",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"pkg": {"internal"},
			},
			DetectUnused: false,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	if len(violations) == 0 {
		t.Fatal("expected violation, got none")
	}

	found := false
	for _, viol := range violations {
		if viol.Type == ViolationPkgToPkg {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected ViolationPkgToPkg")
	}
}

func TestValidate_CrossCmdViolation(t *testing.T) {
	files := []scanner.FileInfo{
		{
			RelPath: "cmd/api/main.go",
			Package: "main",
			Imports: []string{
				"github.com/test/project/cmd/worker",
			},
		},
		{
			RelPath: "cmd/worker/worker.go",
			Package: "worker",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd": {"pkg", "internal"},
			},
			DetectUnused: false,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	if len(violations) == 0 {
		t.Fatal("expected violation, got none")
	}

	found := false
	for _, viol := range violations {
		if viol.Type == ViolationCrossCmd {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected ViolationCrossCmd")
	}
}

func TestValidate_SkipLevelImport(t *testing.T) {
	files := []scanner.FileInfo{
		{
			RelPath: "pkg/orders/service.go",
			Package: "orders",
			Imports: []string{
				"github.com/test/project/pkg/orders/models/entities",
			},
		},
		{
			RelPath: "pkg/orders/models/entities/order.go",
			Package: "entities",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"pkg": {"internal"},
			},
			DetectUnused: false,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	if len(violations) == 0 {
		t.Fatal("expected violation, got none")
	}

	found := false
	for _, viol := range violations {
		if viol.Type == ViolationSkipLevel {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected ViolationSkipLevel")
	}
}

func TestValidate_DirectSubpackageAllowed(t *testing.T) {
	files := []scanner.FileInfo{
		{
			RelPath: "pkg/orders/service.go",
			Package: "orders",
			Imports: []string{
				"github.com/test/project/pkg/orders/models",
			},
		},
		{
			RelPath: "pkg/orders/models/order.go",
			Package: "models",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"pkg": {"internal"},
			},
			DetectUnused: false,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	// Should not have pkg-to-pkg violation for direct subpackage
	for _, viol := range violations {
		if viol.Type == ViolationPkgToPkg || viol.Type == ViolationSkipLevel {
			t.Errorf("unexpected violation: %v", viol.Type)
		}
	}
}

func TestValidate_UnusedPackage(t *testing.T) {
	files := []scanner.FileInfo{
		{
			RelPath: "cmd/api/main.go",
			Package: "main",
			Imports: []string{
				"github.com/test/project/pkg/used",
			},
		},
		{
			RelPath: "pkg/used/service.go",
			Package: "used",
		},
		{
			RelPath: "pkg/unused/service.go",
			Package: "unused",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd": {"pkg", "internal"},
			},
			DetectUnused: true,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	found := false
	for _, viol := range violations {
		if viol.Type == ViolationUnused {
			found = true
			if viol.Issue == "" {
				t.Error("expected issue description")
			}
			break
		}
	}

	if !found {
		t.Error("expected ViolationUnused for pkg/unused")
	}
}

func TestValidate_NoViolations(t *testing.T) {
	files := []scanner.FileInfo{
		{
			RelPath: "cmd/api/main.go",
			Package: "main",
			Imports: []string{
				"github.com/test/project/pkg/service",
				"github.com/test/project/internal/config",
			},
		},
		{
			RelPath: "pkg/service/service.go",
			Package: "service",
			Imports: []string{
				"github.com/test/project/internal/types",
			},
		},
		{
			RelPath: "internal/config/config.go",
			Package: "config",
		},
		{
			RelPath: "internal/types/types.go",
			Package: "types",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd":      {"pkg", "internal"},
				"pkg":      {"internal"},
				"internal": {"internal"},
			},
			DetectUnused: true,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	if len(violations) != 0 {
		t.Errorf("expected no violations, got %d", len(violations))
		for _, viol := range violations {
			t.Logf("  %v: %s", viol.Type, viol.Issue)
		}
	}
}
