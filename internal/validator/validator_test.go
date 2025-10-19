package validator

import (
	"strings"
	"testing"
)

// Test structs that implement validator interfaces for testing

type testConfig struct {
	module                                string
	directoriesImport                     map[string][]string
	detectUnused                          bool
	requiredDirectories                   map[string]string
	allowOtherDirectories                 bool
	detectSharedExternalImports           bool
	sharedExternalImportsMode             string
	sharedExternalImportsExclusions       []string
	sharedExternalImportsExclusionPatterns []string
	lintTestFiles                         bool
	testExemptImports                     []string
	testFileLocation                      string
	requireBlackboxTests                  bool
	coverageEnabled                       bool
	coverageThreshold                     float64
	packageThresholds                     map[string]float64
}

func (tc *testConfig) GetDirectoriesImport() map[string][]string                 { return tc.directoriesImport }
func (tc *testConfig) ShouldDetectUnused() bool                                  { return tc.detectUnused }
func (tc *testConfig) GetRequiredDirectories() map[string]string                 { return tc.requiredDirectories }
func (tc *testConfig) ShouldAllowOtherDirectories() bool                         { return tc.allowOtherDirectories }
func (tc *testConfig) ShouldDetectSharedExternalImports() bool                   { return tc.detectSharedExternalImports }
func (tc *testConfig) GetSharedExternalImportsMode() string                      { return tc.sharedExternalImportsMode }
func (tc *testConfig) GetSharedExternalImportsExclusions() []string              { return tc.sharedExternalImportsExclusions }
func (tc *testConfig) GetSharedExternalImportsExclusionPatterns() []string       { return tc.sharedExternalImportsExclusionPatterns }
func (tc *testConfig) ShouldLintTestFiles() bool                                 { return tc.lintTestFiles }
func (tc *testConfig) GetTestExemptImports() []string                            { return tc.testExemptImports }
func (tc *testConfig) GetTestFileLocation() string                               { return tc.testFileLocation }
func (tc *testConfig) ShouldRequireBlackboxTests() bool                          { return tc.requireBlackboxTests }
func (tc *testConfig) IsCoverageEnabled() bool                                   { return tc.coverageEnabled }
func (tc *testConfig) GetCoverageThreshold() float64                             { return tc.coverageThreshold }
func (tc *testConfig) GetPackageThresholds() map[string]float64 {
	if tc.packageThresholds == nil {
		return make(map[string]float64)
	}
	return tc.packageThresholds
}
func (tc *testConfig) GetModule() string { return tc.module }

type testDependency struct {
	importPath string
	localPath  string
	isLocal    bool
}

func (td *testDependency) GetImportPath() string { return td.importPath }
func (td *testDependency) GetLocalPath() string  { return td.localPath }
func (td *testDependency) IsLocalDep() bool      { return td.isLocal }

type testFileNode struct {
	relPath      string
	pkg          string
	dependencies []Dependency
}

func (tfn *testFileNode) GetRelPath() string           { return tfn.relPath }
func (tfn *testFileNode) GetPackage() string           { return tfn.pkg }
func (tfn *testFileNode) GetDependencies() []Dependency { return tfn.dependencies }

type testGraph struct {
	nodes []FileNode
}

func (tg *testGraph) GetNodes() []FileNode { return tg.nodes }

func TestValidate_PkgToPkgViolation(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "pkg/http/server.go",
				pkg:     "http",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/pkg/database", localPath: "pkg/database", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "pkg/database/db.go",
				pkg:          "database",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"pkg": {"internal"},
		},
		detectUnused: false,
	}

	v := New(cfg, g)
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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/api/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/cmd/worker", localPath: "cmd/worker", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "cmd/worker/worker.go",
				pkg:          "worker",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"cmd": {"pkg", "internal"},
		},
		detectUnused: false,
	}

	v := New(cfg, g)
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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "pkg/orders/service.go",
				pkg:     "orders",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/pkg/orders/models/entities", localPath: "pkg/orders/models/entities", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "pkg/orders/models/entities/order.go",
				pkg:          "entities",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"pkg": {"internal"},
		},
		detectUnused: false,
	}

	v := New(cfg, g)
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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "pkg/orders/service.go",
				pkg:     "orders",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/pkg/orders/models", localPath: "pkg/orders/models", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "pkg/orders/models/order.go",
				pkg:          "models",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"pkg": {"internal"},
		},
		detectUnused: false,
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should not have pkg-to-pkg violation for direct subpackage
	for _, viol := range violations {
		if viol.Type == ViolationPkgToPkg || viol.Type == ViolationSkipLevel {
			t.Errorf("unexpected violation: %v", viol.Type)
		}
	}
}

func TestValidate_UnusedPackage(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/api/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/pkg/used", localPath: "pkg/used", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "pkg/used/service.go",
				pkg:          "used",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "pkg/unused/service.go",
				pkg:          "unused",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"cmd": {"pkg", "internal"},
		},
		detectUnused: true,
	}

	v := New(cfg, g)
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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "internal/output/markdown.go",
				pkg:     "output",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/graph", localPath: "internal/graph", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "internal/graph/graph.go",
				pkg:          "graph",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"cmd":      {"pkg"},
			"pkg":      {"internal"},
			"internal": {}, // Empty array - internal packages cannot import anything
		},
		detectUnused: false,
	}

	v := New(cfg, g)
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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/api/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/pkg/service", localPath: "pkg/service", isLocal: true},
					&testDependency{importPath: "github.com/test/project/internal/config", localPath: "internal/config", isLocal: true},
				},
			},
			&testFileNode{
				relPath: "pkg/service/service.go",
				pkg:     "service",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/types", localPath: "internal/types", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "internal/config/config.go",
				pkg:          "config",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "internal/types/types.go",
				pkg:          "types",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"cmd":      {"pkg", "internal"},
			"pkg":      {"internal"},
			"internal": {"internal"},
		},
		detectUnused: true,
	}

	v := New(cfg, g)
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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/dw/claude.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/app", localPath: "internal/app", isLocal: true},
					&testDependency{importPath: "github.com/test/project/internal/infra", localPath: "internal/infra", isLocal: true},
				},
			},
			&testFileNode{
				relPath: "cmd/dw/logs.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/app", localPath: "internal/app", isLocal: true},
				},
			},
			&testFileNode{
				relPath: "internal/app/app.go",
				pkg:     "app",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/domain", localPath: "internal/domain", isLocal: true},
				},
			},
			&testFileNode{
				relPath: "internal/infra/db.go",
				pkg:     "infra",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/domain", localPath: "internal/domain", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "internal/domain/user.go",
				pkg:          "domain",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"cmd":              {"internal/app", "internal/infra"},
			"cmd/dw":           {"internal/app", "internal/infra"}, // More specific rule
			"internal/app":     {"internal/domain"},
			"internal/domain":  {},
			"internal/infra":   {"internal/domain"},
		},
		detectUnused: false,
	}

	v := New(cfg, g)
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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/api/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/app/user", localPath: "internal/app/user", isLocal: true},
					&testDependency{importPath: "github.com/test/project/internal/infra/database", localPath: "internal/infra/database", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "internal/app/user/service.go",
				pkg:          "user",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "internal/infra/database/conn.go",
				pkg:          "database",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"cmd": {"internal/app", "internal/infra"}, // Allows subpackages too
		},
		detectUnused: false,
	}

	v := New(cfg, g)
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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/api/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/forbidden", localPath: "internal/forbidden", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "internal/forbidden/service.go",
				pkg:          "forbidden",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"cmd": {"internal/app", "internal/infra"}, // Does NOT include "internal/forbidden"
		},
		detectUnused: false,
	}

	v := New(cfg, g)
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
	// Create graph with cmd and internal both importing github.com/pkg/errors (external non-stdlib)
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/pkg/errors", isLocal: false},
					&testDependency{importPath: "fmt", isLocal: false},
				},
			},
			&testFileNode{
				relPath: "internal/repo.go",
				pkg:     "repo",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/pkg/errors", isLocal: false},
				},
			},
		},
	}

	cfg := &testConfig{
		module: "example.com/test",
		directoriesImport: map[string][]string{
			"cmd":      {"pkg"},
			"internal": {},
		},
		detectSharedExternalImports: true,
		sharedExternalImportsMode:   "error",
	}

	v := New(cfg, g)
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
	// Create graph with cmd and internal both importing github.com/pkg/errors (excluded)
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/pkg/errors", isLocal: false},
				},
			},
			&testFileNode{
				relPath: "internal/repo.go",
				pkg:     "repo",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/pkg/errors", isLocal: false},
				},
			},
		},
	}

	cfg := &testConfig{
		module: "example.com/test",
		directoriesImport: map[string][]string{
			"cmd":      {"pkg"},
			"internal": {},
		},
		detectSharedExternalImports:        true,
		sharedExternalImportsMode:          "error",
		sharedExternalImportsExclusions:    []string{"fmt", "github.com/pkg/errors"},
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should NOT have violation for github.com/pkg/errors (it's excluded)
	for _, viol := range violations {
		if viol.Type == ViolationSharedExternalImport && strings.Contains(viol.Issue, "github.com/pkg/errors") {
			t.Error("Expected no violation for github.com/pkg/errors (excluded)")
		}
	}
}

func TestDetectSharedExternalImports_GlobExclusion(t *testing.T) {
	// Create graph with cmd and internal both importing encoding/json (should be excluded)
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "encoding/json", isLocal: false},
				},
			},
			&testFileNode{
				relPath: "internal/repo.go",
				pkg:     "repo",
				dependencies: []Dependency{
					&testDependency{importPath: "encoding/json", isLocal: false},
				},
			},
		},
	}

	cfg := &testConfig{
		module: "example.com/test",
		directoriesImport: map[string][]string{
			"cmd":      {"pkg"},
			"internal": {},
		},
		detectSharedExternalImports:               true,
		sharedExternalImportsMode:                 "error",
		sharedExternalImportsExclusionPatterns:    []string{"encoding/*"},
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should NOT have violation for encoding/json (matches pattern)
	for _, viol := range violations {
		if viol.Type == ViolationSharedExternalImport && strings.Contains(viol.Issue, "encoding/json") {
			t.Error("Expected no violation for encoding/json (matches encoding/* pattern)")
		}
	}
}

func TestDetectSharedExternalImports_SingleLayerNoViolation(t *testing.T) {
	// Create graph with multiple files in same layer (internal) importing same package
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "internal/repo.go",
				pkg:     "repo",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/pkg/errors", isLocal: false},
				},
			},
			&testFileNode{
				relPath: "internal/store.go",
				pkg:     "store",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/pkg/errors", isLocal: false},
				},
			},
		},
	}

	cfg := &testConfig{
		module: "example.com/test",
		directoriesImport: map[string][]string{
			"cmd":      {"pkg"},
			"internal": {},
		},
		detectSharedExternalImports: true,
		sharedExternalImportsMode:   "error",
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should NOT have violation (same layer is OK)
	for _, viol := range violations {
		if viol.Type == ViolationSharedExternalImport {
			t.Errorf("Expected no shared external import violations, got: %v", viol)
		}
	}
}

// TestBlackBoxTest_ParentPackageImport tests that black-box tests can import their parent package
func TestBlackBoxTest_ParentPackageImport(t *testing.T) {
	// Black-box test file importing its parent package
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/analysis.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath: "internal/app/analysis_test.go",
				pkg:     "app_test", // Black-box test (package name ends with _test)
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/app", localPath: "internal/app", isLocal: true},
				},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"internal": {}, // internal packages cannot import each other
		},
		detectUnused:  false,
		lintTestFiles: true,
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should NOT have any violations - black-box tests are allowed to import parent package
	if len(violations) > 0 {
		t.Errorf("Expected no violations for black-box test importing parent package, got %d violations:", len(violations))
		for _, viol := range violations {
			t.Logf("  - Type: %s, File: %s, Issue: %s", viol.Type, viol.File, viol.Issue)
		}
	}
}

// TestWhiteBoxTest_NormalRules tests that white-box tests follow normal architectural rules
func TestWhiteBoxTest_NormalRules(t *testing.T) {
	// White-box test file (same package name, not ending with _test)
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/analysis.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath: "internal/app/analysis_test.go",
				pkg:     "app", // White-box test (same package, no _test suffix)
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/config", localPath: "internal/config", isLocal: true},
				},
			},
			&testFileNode{
				relPath:      "internal/config/config.go",
				pkg:          "config",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"internal": {}, // internal packages cannot import each other
		},
		detectUnused:  false,
		lintTestFiles: true,
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should HAVE violations - white-box tests follow normal rules
	if len(violations) == 0 {
		t.Fatal("Expected violations for white-box test importing another internal package, got none")
	}

	found := false
	for _, viol := range violations {
		if viol.Type == ViolationForbidden && strings.Contains(viol.File, "analysis_test.go") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected ViolationForbidden for white-box test importing another internal package")
	}
}

// TestBlackBoxTest_OtherImportsFollowNormalRules tests that black-box tests' other imports follow normal architecture rules
func TestBlackBoxTest_OtherImportsFollowNormalRules(t *testing.T) {
	// Black-box test importing both parent package (exempted) and another internal package (subject to normal rules)
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/analysis.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath: "internal/app/analysis_test.go",
				pkg:     "app_test", // Black-box test
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/internal/app", localPath: "internal/app", isLocal: true},       // Parent package - exempted
					&testDependency{importPath: "github.com/test/project/internal/config", localPath: "internal/config", isLocal: true}, // Other internal package - follows normal rules (forbidden by internal:[])
				},
			},
			&testFileNode{
				relPath:      "internal/config/config.go",
				pkg:          "config",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"internal": {}, // internal packages cannot import each other
		},
		detectUnused:  false,
		lintTestFiles: true,
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should have ViolationForbidden for importing internal/config (normal architecture rule)
	if len(violations) == 0 {
		t.Fatal("Expected violation for black-box test importing another internal package (violates internal:[] rule), got none")
	}

	foundConfigViolation := false
	foundAppAsTargetViolation := false

	for _, viol := range violations {
		if strings.Contains(viol.File, "analysis_test.go") {
			// Check for forbidden import violation about importing config
			if viol.Type == ViolationForbidden && strings.Contains(viol.Issue, "internal/config") {
				foundConfigViolation = true
			}
			// Check if there's a violation where internal/app is the TARGET of the import
			if strings.Contains(viol.Issue, "imports internal/app") {
				foundAppAsTargetViolation = true
			}
		}
	}

	if !foundConfigViolation {
		t.Error("Expected ViolationForbidden for importing internal/config (normal architecture rule)")
	}

	if foundAppAsTargetViolation {
		t.Error("Did not expect violation for importing parent package internal/app (should be exempted)")
	}
}

// TestBlackBoxTest_InPkgLayer tests black-box tests in pkg layer
func TestBlackBoxTest_InPkgLayer(t *testing.T) {
	// Black-box test in pkg layer importing parent package
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "pkg/linter/linter.go",
				pkg:          "linter",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath: "pkg/linter/linter_test.go",
				pkg:     "linter_test", // Black-box test
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/pkg/linter", localPath: "pkg/linter", isLocal: true},
				},
			},
		},
	}

	cfg := &testConfig{
		module: "github.com/test/project",
		directoriesImport: map[string][]string{
			"pkg": {"internal"}, // pkg can import internal
		},
		detectUnused:  false,
		lintTestFiles: true,
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should NOT have violations - black-box tests can import parent package
	if len(violations) > 0 {
		t.Errorf("Expected no violations for black-box test in pkg layer importing parent package, got %d:", len(violations))
		for _, viol := range violations {
			t.Logf("  - Type: %s, File: %s, Issue: %s", viol.Type, viol.File, viol.Issue)
		}
	}
}

// TestIsBlackBoxTest tests the isBlackBoxTest helper function
func TestIsBlackBoxTest(t *testing.T) {
	cfg := &testConfig{}
	v := New(cfg, &testGraph{})

	tests := []struct {
		name        string
		relPath     string
		packageName string
		want        bool
	}{
		{
			name:        "black-box test",
			relPath:     "internal/app/analysis_test.go",
			packageName: "app_test",
			want:        true,
		},
		{
			name:        "white-box test",
			relPath:     "internal/app/analysis_test.go",
			packageName: "app",
			want:        false,
		},
		{
			name:        "non-test file with _test package",
			relPath:     "internal/app/analysis.go",
			packageName: "app_test",
			want:        false,
		},
		{
			name:        "non-test file",
			relPath:     "internal/app/analysis.go",
			packageName: "app",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &testFileNode{
				relPath: tt.relPath,
				pkg:     tt.packageName,
			}
			got := v.isBlackBoxTest(node)
			if got != tt.want {
				t.Errorf("isBlackBoxTest() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsParentPackageImport tests the isParentPackageImport helper function
func TestIsParentPackageImport(t *testing.T) {
	cfg := &testConfig{}
	v := New(cfg, &testGraph{})

	tests := []struct {
		name       string
		fileDir    string
		importPath string
		want       bool
	}{
		{
			name:       "parent package import",
			fileDir:    "internal/app",
			importPath: "internal/app",
			want:       true,
		},
		{
			name:       "non-parent import",
			fileDir:    "internal/app",
			importPath: "internal/config",
			want:       false,
		},
		{
			name:       "nested package import",
			fileDir:    "internal/app",
			importPath: "internal/app/models",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.isParentPackageImport(tt.fileDir, tt.importPath)
			if got != tt.want {
				t.Errorf("isParentPackageImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValidateTestFileLocations_Colocated tests that colocated policy requires tests next to code
func TestValidateTestFileLocations_Colocated(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/app.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "internal/app/app_test.go", // ✓ Colocated - OK
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "tests/internal/config/config_test.go", // ✗ In tests/ directory - violation
				pkg:          "config",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module:           "github.com/test/project",
		lintTestFiles:    true,
		testFileLocation: "colocated", // Tests must be next to code
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should have violation for test in tests/ directory
	foundViolation := false
	for _, viol := range violations {
		if viol.Type == ViolationTestFileLocation && strings.Contains(viol.File, "tests/") {
			foundViolation = true
			break
		}
	}

	if !foundViolation {
		t.Error("Expected ViolationTestFileLocation for test file in tests/ directory with colocated policy")
	}

	// Should NOT have violation for colocated test
	for _, viol := range violations {
		if viol.Type == ViolationTestFileLocation && strings.Contains(viol.File, "internal/app/app_test.go") {
			t.Error("Did not expect violation for colocated test file")
		}
	}
}

// TestValidateTestFileLocations_Separate tests that separate policy requires tests in tests/ directory
func TestValidateTestFileLocations_Separate(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/app.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "internal/app/app_test.go", // ✗ Colocated - violation
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "tests/internal/config/config_test.go", // ✓ In tests/ directory - OK
				pkg:          "config",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module:           "github.com/test/project",
		lintTestFiles:    true,
		testFileLocation: "separate", // Tests must be in tests/ directory
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should have violation for colocated test
	foundViolation := false
	for _, viol := range violations {
		if viol.Type == ViolationTestFileLocation && strings.Contains(viol.File, "internal/app/app_test.go") {
			foundViolation = true
			break
		}
	}

	if !foundViolation {
		t.Error("Expected ViolationTestFileLocation for colocated test file with separate policy")
	}

	// Should NOT have violation for test in tests/ directory
	for _, viol := range violations {
		if viol.Type == ViolationTestFileLocation && strings.Contains(viol.File, "tests/") {
			t.Error("Did not expect violation for test file in tests/ directory")
		}
	}
}

// TestValidateTestFileLocations_Any tests that "any" policy allows tests anywhere
func TestValidateTestFileLocations_Any(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/app.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "internal/app/app_test.go", // ✓ Colocated - OK
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "tests/internal/config/config_test.go", // ✓ In tests/ - OK
				pkg:          "config",
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module:           "github.com/test/project",
		lintTestFiles:    true,
		testFileLocation: "any", // Tests can be anywhere
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should NOT have any test location violations
	for _, viol := range violations {
		if viol.Type == ViolationTestFileLocation {
			t.Errorf("Did not expect ViolationTestFileLocation with 'any' policy, got: %v", viol)
		}
	}
}

// TestValidateBlackboxTests_WhiteboxDetected tests that whitebox tests are detected
func TestValidateBlackboxTests_WhiteboxDetected(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/app.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "internal/app/app_test.go", // Whitebox test
				pkg:          "app",                      // Should be "app_test"
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "pkg/database/db_test.go", // Whitebox test
				pkg:          "database",                // Should be "database_test"
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module:               "github.com/test/project",
		requireBlackboxTests: true, // Enable blackbox test requirement
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should have 2 whitebox test violations
	whiteboxViolations := 0
	for _, viol := range violations {
		if viol.Type == ViolationWhiteboxTest {
			whiteboxViolations++
		}
	}

	if whiteboxViolations != 2 {
		t.Errorf("Expected 2 whitebox test violations, got %d", whiteboxViolations)
	}

	// Check violation details
	foundAppViolation := false
	foundDbViolation := false
	for _, viol := range violations {
		if viol.Type == ViolationWhiteboxTest {
			if strings.Contains(viol.File, "internal/app/app_test.go") {
				foundAppViolation = true
				if !strings.Contains(viol.Issue, "package app instead of app_test") {
					t.Errorf("Expected issue to mention 'package app instead of app_test', got: %s", viol.Issue)
				}
			}
			if strings.Contains(viol.File, "pkg/database/db_test.go") {
				foundDbViolation = true
				if !strings.Contains(viol.Issue, "package database instead of database_test") {
					t.Errorf("Expected issue to mention 'package database instead of database_test', got: %s", viol.Issue)
				}
			}
		}
	}

	if !foundAppViolation {
		t.Error("Expected whitebox violation for internal/app/app_test.go")
	}
	if !foundDbViolation {
		t.Error("Expected whitebox violation for pkg/database/db_test.go")
	}

	// Verify violations are concise (educational content is in separate guidance section)
	for _, viol := range violations {
		if viol.Type == ViolationWhiteboxTest {
			// Check that Rule is concise
			if !strings.Contains(viol.Rule, "Blackbox testing is enforced") {
				t.Errorf("Expected Rule to mention blackbox testing requirement, got: %s", viol.Rule)
			}

			// Rule should NOT contain educational content (that's in guidance section)
			if strings.Contains(viol.Rule, "WHY THIS MATTERS") {
				t.Error("Expected Rule to be concise, not contain 'WHY THIS MATTERS' - that belongs in guidance section")
			}

			// Check that Fix is concise
			if !strings.Contains(viol.Fix, "Change package declaration") {
				t.Errorf("Expected Fix to mention changing package declaration, got: %s", viol.Fix)
			}

			// Fix should NOT contain detailed steps (that's in guidance section)
			if strings.Contains(viol.Fix, "After changing to blackbox testing") {
				t.Error("Expected Fix to be concise, not contain detailed guidance")
			}
		}
	}
}

// TestValidateBlackboxTests_BlackboxAllowed tests that blackbox tests are allowed
func TestValidateBlackboxTests_BlackboxAllowed(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/app.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "internal/app/app_test.go", // Blackbox test
				pkg:          "app_test",                 // Correctly uses _test suffix
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "pkg/database/db_test.go", // Blackbox test
				pkg:          "database_test",           // Correctly uses _test suffix
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module:               "github.com/test/project",
		requireBlackboxTests: true, // Enable blackbox test requirement
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should NOT have any whitebox test violations
	for _, viol := range violations {
		if viol.Type == ViolationWhiteboxTest {
			t.Errorf("Did not expect ViolationWhiteboxTest for blackbox tests, got: %v", viol)
		}
	}
}

// TestValidateBlackboxTests_DisabledByDefault tests that the rule is disabled when not configured
func TestValidateBlackboxTests_DisabledByDefault(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "internal/app/app.go",
				pkg:          "app",
				dependencies: []Dependency{},
			},
			&testFileNode{
				relPath:      "internal/app/app_test.go", // Whitebox test
				pkg:          "app",                      // Should be "app_test" if rule enabled
				dependencies: []Dependency{},
			},
		},
	}

	cfg := &testConfig{
		module:               "github.com/test/project",
		requireBlackboxTests: false, // Rule disabled
	}

	v := New(cfg, g)
	violations := v.Validate()

	// Should NOT have any whitebox test violations when rule is disabled
	for _, viol := range violations {
		if viol.Type == ViolationWhiteboxTest {
			t.Errorf("Did not expect ViolationWhiteboxTest when rule is disabled, got: %v", viol)
		}
	}
}
