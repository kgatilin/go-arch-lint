package output

import (
	"strings"
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/graph"
	"github.com/kgatilin/go-arch-lint/internal/scanner"
	"github.com/kgatilin/go-arch-lint/internal/validator"
)

// Helper to convert []scanner.FileInfo to []graph.FileInfo (slice covariance workaround)
func toGraphFiles(files []scanner.FileInfo) []graph.FileInfo {
	result := make([]graph.FileInfo, len(files))
	for i := range files {
		result[i] = files[i]
	}
	return result
}

// Test adapter to convert graph.Graph to output.Graph interface
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

func (tfna *testFileNodeAdapter) GetPackage() string {
	return tfna.node.Package
}

func (tfna *testFileNodeAdapter) GetDependencies() []Dependency {
	deps := make([]Dependency, len(tfna.node.Dependencies))
	for i := range tfna.node.Dependencies {
		deps[i] = &tfna.node.Dependencies[i]
	}
	return deps
}

// Test adapter to convert validator.Violation to output.Violation interface
type testViolationAdapter struct {
	v *validator.Violation
}

func (tva *testViolationAdapter) GetType() string   { return string(tva.v.Type) }
func (tva *testViolationAdapter) GetFile() string   { return tva.v.File }
func (tva *testViolationAdapter) GetLine() int      { return tva.v.Line }
func (tva *testViolationAdapter) GetIssue() string  { return tva.v.Issue }
func (tva *testViolationAdapter) GetRule() string   { return tva.v.Rule }
func (tva *testViolationAdapter) GetFix() string    { return tva.v.Fix }

func TestGenerateMarkdown_Basic(t *testing.T) {
	files := []scanner.FileInfo{
		{
			RelPath: "cmd/api/main.go",
			Package: "main",
			Imports: []string{
				"fmt",
				"github.com/test/project/pkg/service",
			},
		},
		{
			RelPath: "pkg/service/service.go",
			Package: "service",
			Imports: []string{
				"github.com/external/lib",
			},
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")
	md := GenerateMarkdown(&testGraphAdapter{g: g})

	// Check header
	if !strings.Contains(md, "# Dependency Graph") {
		t.Error("missing header")
	}

	// Check file sections
	if !strings.Contains(md, "## cmd/api/main.go") {
		t.Error("missing cmd/api/main.go section")
	}

	if !strings.Contains(md, "## pkg/service/service.go") {
		t.Error("missing pkg/service/service.go section")
	}

	// Check local dependency
	if !strings.Contains(md, "local:pkg/service") {
		t.Error("missing local dependency")
	}

	// Check external dependency
	if !strings.Contains(md, "external:github.com/external/lib") {
		t.Error("missing external dependency")
	}

	// Standard library should not appear
	if strings.Contains(md, "fmt") {
		t.Error("standard library should not appear in output")
	}
}

func TestGenerateMarkdown_NoDependencies(t *testing.T) {
	files := []scanner.FileInfo{
		{
			RelPath: "pkg/types/types.go",
			Package: "types",
			Imports: []string{},
		},
	}

	g := graph.Build(toGraphFiles(files), "github.com/test/project")
	md := GenerateMarkdown(&testGraphAdapter{g: g})

	if !strings.Contains(md, "depends on: (none)") {
		t.Error("expected 'depends on: (none)' for file with no dependencies")
	}
}

func TestFormatViolations_NoViolations(t *testing.T) {
	var violations []Violation
	result := FormatViolations(violations)

	if result != "" {
		t.Error("expected empty string for no violations")
	}
}

func TestFormatViolations_WithViolations(t *testing.T) {
	viol1 := validator.Violation{
		Type:  validator.ViolationPkgToPkg,
		File:  "pkg/http/handler.go",
		Issue: "pkg/http imports pkg/database",
		Rule:  "pkg packages must not import other pkg packages",
		Fix:   "Import from internal/ or define interface locally",
	}
	viol2 := validator.Violation{
		Type:  validator.ViolationCrossCmd,
		File:  "cmd/api/main.go",
		Line:  10,
		Issue: "cmd/api imports cmd/worker",
		Rule:  "cmd packages must not import other cmd packages",
		Fix:   "Extract shared code to pkg/ or internal/",
	}

	violations := []Violation{
		&testViolationAdapter{v: &viol1},
		&testViolationAdapter{v: &viol2},
	}

	result := FormatViolations(violations)

	// Check header
	if !strings.Contains(result, "DEPENDENCY VIOLATIONS DETECTED") {
		t.Error("missing violations header")
	}

	// Check first violation
	if !strings.Contains(result, "[ERROR] Forbidden pkg-to-pkg Dependency") {
		t.Error("missing first violation type")
	}

	if !strings.Contains(result, "File: pkg/http/handler.go") {
		t.Error("missing first violation file")
	}

	if !strings.Contains(result, "Issue: pkg/http imports pkg/database") {
		t.Error("missing first violation issue")
	}

	if !strings.Contains(result, "Rule: pkg packages must not import other pkg packages") {
		t.Error("missing first violation rule")
	}

	if !strings.Contains(result, "Fix: Import from internal/ or define interface locally") {
		t.Error("missing first violation fix")
	}

	// Check second violation with line number
	if !strings.Contains(result, "[ERROR] Cross-cmd Dependency") {
		t.Error("missing second violation type")
	}

	if !strings.Contains(result, "File: cmd/api/main.go:10") {
		t.Error("missing second violation file with line number")
	}
}

func TestFormatViolations_UnusedPackage(t *testing.T) {
	viol := validator.Violation{
		Type:  validator.ViolationUnused,
		Issue: "Package pkg/legacy not imported by any cmd/ package",
		Rule:  "All packages should be transitively imported from cmd/",
		Fix:   "Remove package or add import from cmd/",
	}

	violations := []Violation{
		&testViolationAdapter{v: &viol},
	}

	result := FormatViolations(violations)

	// Unused packages don't have a file
	if strings.Contains(result, "File:") {
		t.Error("unused package violation should not have File field")
	}

	if !strings.Contains(result, "[ERROR] Unused Package") {
		t.Error("missing violation type")
	}

	if !strings.Contains(result, "Issue: Package pkg/legacy not imported") {
		t.Error("missing issue")
	}
}
