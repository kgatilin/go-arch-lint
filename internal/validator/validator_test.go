package validator

import (
	"strings"
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

func TestValidate_InternalToInternalViolation(t *testing.T) {
	// Regression test for bug where internal: [] did not catch internal-to-internal imports
	files := []scanner.FileInfo{
		{
			RelPath: "internal/output/markdown.go",
			Package: "output",
			Imports: []string{
				"github.com/test/project/internal/graph",
			},
		},
		{
			RelPath: "internal/graph/graph.go",
			Package: "graph",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd":      {"pkg"},
				"pkg":      {"internal"},
				"internal": {}, // Empty array - internal packages cannot import anything
			},
			DetectUnused: false,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	if len(violations) == 0 {
		t.Fatal("expected violation for internal-to-internal import with internal: [], got none")
	}

	found := false
	for _, viol := range violations {
		if viol.Type == ViolationForbidden && viol.File == "internal/output/markdown.go" {
			found = true
			if viol.Rule != "internal can only import from: []" {
				t.Errorf("expected rule 'internal can only import from: []', got %q", viol.Rule)
			}
			if viol.Fix != "Use interfaces and dependency inversion instead of direct imports" {
				t.Errorf("expected specific fix message for internal-to-internal, got %q", viol.Fix)
			}
			break
		}
	}

	if !found {
		t.Error("expected ViolationForbidden for internal/output importing internal/graph")
		for _, viol := range violations {
			t.Logf("  got: %v - %s", viol.Type, viol.Issue)
		}
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

func TestValidate_SubdirectorySpecificRule(t *testing.T) {
	// Regression test for bug where subdirectory-specific rules (cmd/dw) were ignored
	files := []scanner.FileInfo{
		{
			RelPath: "cmd/dw/claude.go",
			Package: "main",
			Imports: []string{
				"github.com/test/project/internal/app",
				"github.com/test/project/internal/infra",
			},
		},
		{
			RelPath: "cmd/dw/logs.go",
			Package: "main",
			Imports: []string{
				"github.com/test/project/internal/app",
			},
		},
		{
			RelPath: "internal/app/app.go",
			Package: "app",
			Imports: []string{
				"github.com/test/project/internal/domain",
			},
		},
		{
			RelPath: "internal/infra/db.go",
			Package: "infra",
			Imports: []string{
				"github.com/test/project/internal/domain",
			},
		},
		{
			RelPath: "internal/domain/user.go",
			Package: "domain",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd": {"internal/app", "internal/infra"},
				"cmd/dw": {"internal/app", "internal/infra"}, // More specific rule
				"internal/app": {"internal/domain"},
				"internal/domain": {},
				"internal/infra": {"internal/domain"},
			},
			DetectUnused: false,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	// Should not have any violations - the subdirectory-specific rule should allow these imports
	for _, viol := range violations {
		if viol.Type == ViolationForbidden && viol.File == "cmd/dw/claude.go" {
			t.Errorf("unexpected ViolationForbidden for cmd/dw/claude.go: %s", viol.Issue)
		}
		if viol.Type == ViolationForbidden && viol.File == "cmd/dw/logs.go" {
			t.Errorf("unexpected ViolationForbidden for cmd/dw/logs.go: %s", viol.Issue)
		}
	}

	if len(violations) != 0 {
		t.Logf("violations found:")
		for _, viol := range violations {
			t.Logf("  %v: %s in %s", viol.Type, viol.Issue, viol.File)
		}
	}
}

func TestValidate_PrefixMatchingForAllowedImports(t *testing.T) {
	// Test that if "internal/app" is allowed, then "internal/app/user" is also allowed
	files := []scanner.FileInfo{
		{
			RelPath: "cmd/api/main.go",
			Package: "main",
			Imports: []string{
				"github.com/test/project/internal/app/user",
				"github.com/test/project/internal/infra/database",
			},
		},
		{
			RelPath: "internal/app/user/service.go",
			Package: "user",
		},
		{
			RelPath: "internal/infra/database/conn.go",
			Package: "database",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd": {"internal/app", "internal/infra"}, // Allows subpackages too
			},
			DetectUnused: false,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	// Should not have any violations - prefix matching should allow these imports
	for _, viol := range violations {
		if viol.Type == ViolationForbidden {
			t.Errorf("unexpected ViolationForbidden: %s - %s", viol.File, viol.Issue)
		}
	}
}

func TestValidate_ForbiddenImportNotInAllowedList(t *testing.T) {
	// Ensure that imports NOT in the allowed list are still caught
	files := []scanner.FileInfo{
		{
			RelPath: "cmd/api/main.go",
			Package: "main",
			Imports: []string{
				"github.com/test/project/internal/forbidden",
			},
		},
		{
			RelPath: "internal/forbidden/service.go",
			Package: "forbidden",
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")

	cfg := &config.Config{
		Module: "github.com/test/project",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd": {"internal/app", "internal/infra"}, // Does NOT include "internal/forbidden"
			},
			DetectUnused: false,
		},
	}

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	found := false
	for _, viol := range violations {
		if viol.Type == ViolationForbidden && viol.File == "cmd/api/main.go" {
			found = true
			if viol.Rule != "cmd can only import from: [internal/app internal/infra]" {
				t.Errorf("expected specific rule message, got %q", viol.Rule)
			}
			break
		}
	}

	if !found {
		t.Error("expected ViolationForbidden for cmd importing internal/forbidden")
	}
}

func TestDetectSharedExternalImports_MultipleLayersImportSamePackage(t *testing.T) {
	// Create config with shared_external_imports enabled
	cfg := &config.Config{
		Module: "example.com/test",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd":      {"pkg"},
				"internal": {},
			},
			SharedExternalImports: config.SharedExternalImports{
				Mode:   "error",
				Detect: true,
			},
		},
	}

	// Create graph with cmd and internal both importing github.com/pkg/errors (external non-stdlib)
	files := []scanner.FileInfo{
		{
			Path:    "/project/cmd/main.go",
			RelPath: "cmd/main.go",
			Package: "main",
			Imports: []string{"github.com/pkg/errors", "fmt"},
		},
		{
			Path:    "/project/internal/repo.go",
			RelPath: "internal/repo.go",
			Package: "repo",
			Imports: []string{"github.com/pkg/errors"},
		},
	}

	module := "example.com/test"
	g := graph.Build(toGraphFiles(files), module)

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	// Should have exactly 1 violation for github.com/pkg/errors
	found := false
	for _, viol := range violations {
		if viol.Type == ViolationSharedExternalImport && strings.Contains(viol.Issue, "github.com/pkg/errors") {
			found = true
			// Verify violation details
			if !strings.Contains(viol.Issue, "2 layers") {
				t.Errorf("Expected '2 layers' in issue, got: %s", viol.Issue)
			}
		}
	}

	if !found {
		t.Error("Expected ViolationSharedExternalImport for github.com/pkg/errors")
	}
}

func TestDetectSharedExternalImports_ExactExclusion(t *testing.T) {
	// Create config with github.com/pkg/errors in exclusions
	cfg := &config.Config{
		Module: "example.com/test",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd":      {"pkg"},
				"internal": {},
			},
			SharedExternalImports: config.SharedExternalImports{
				Detect:     true,
				Mode:       "error",
				Exclusions: []string{"fmt", "github.com/pkg/errors"},
			},
		},
	}

	// Create graph with cmd and internal both importing github.com/pkg/errors (excluded)
	files := []scanner.FileInfo{
		{
			Path:    "/project/cmd/main.go",
			RelPath: "cmd/main.go",
			Package: "main",
			Imports: []string{"github.com/pkg/errors"},
		},
		{
			Path:    "/project/internal/repo.go",
			RelPath: "internal/repo.go",
			Package: "repo",
			Imports: []string{"github.com/pkg/errors"},
		},
	}

	module := "example.com/test"
	g := graph.Build(toGraphFiles(files), module)

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	// Should NOT have violation for github.com/pkg/errors (it's excluded)
	for _, viol := range violations {
		if viol.Type == ViolationSharedExternalImport && strings.Contains(viol.Issue, "github.com/pkg/errors") {
			t.Error("Expected no violation for github.com/pkg/errors (excluded)")
		}
	}
}

func TestDetectSharedExternalImports_GlobExclusion(t *testing.T) {
	// Create config with encoding/* pattern
	cfg := &config.Config{
		Module: "example.com/test",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd":      {"pkg"},
				"internal": {},
			},
			SharedExternalImports: config.SharedExternalImports{
				Detect:            true,
				Mode:              "error",
				ExclusionPatterns: []string{"encoding/*"},
			},
		},
	}

	// Create graph with cmd and internal both importing encoding/json (should be excluded)
	files := []scanner.FileInfo{
		{
			Path:    "/project/cmd/main.go",
			RelPath: "cmd/main.go",
			Package: "main",
			Imports: []string{"encoding/json"},
		},
		{
			Path:    "/project/internal/repo.go",
			RelPath: "internal/repo.go",
			Package: "repo",
			Imports: []string{"encoding/json"},
		},
	}

	module := "example.com/test"
	g := graph.Build(toGraphFiles(files), module)

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	// Should NOT have violation for encoding/json (matches pattern)
	for _, viol := range violations {
		if viol.Type == ViolationSharedExternalImport && strings.Contains(viol.Issue, "encoding/json") {
			t.Error("Expected no violation for encoding/json (matches encoding/* pattern)")
		}
	}
}

func TestDetectSharedExternalImports_SingleLayerNoViolation(t *testing.T) {
	// Create config
	cfg := &config.Config{
		Module: "example.com/test",
		Rules: config.Rules{
			DirectoriesImport: map[string][]string{
				"cmd":      {"pkg"},
				"internal": {},
			},
			SharedExternalImports: config.SharedExternalImports{
				Detect: true,
				Mode:   "error",
			},
		},
	}

	// Create graph with multiple files in same layer (internal) importing same package
	files := []scanner.FileInfo{
		{
			Path:    "/project/internal/repo.go",
			RelPath: "internal/repo.go",
			Package: "repo",
			Imports: []string{"github.com/pkg/errors"},
		},
		{
			Path:    "/project/internal/store.go",
			RelPath: "internal/store.go",
			Package: "store",
			Imports: []string{"github.com/pkg/errors"},
		},
	}

	module := "example.com/test"
	g := graph.Build(toGraphFiles(files), module)

	v := New(cfg, &testGraphAdapter{g: g})
	violations := v.Validate()

	// Should NOT have violation (same layer is OK)
	for _, viol := range violations {
		if viol.Type == ViolationSharedExternalImport {
			t.Errorf("Expected no shared external import violations, got: %v", viol)
		}
	}
}
