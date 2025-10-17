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

// Test adapter for FileWithAPI
type testFileWithAPI struct {
	relPath string
	pkg     string
	decls   []ExportedDecl
}

func (tf *testFileWithAPI) GetRelPath() string {
	return tf.relPath
}

func (tf *testFileWithAPI) GetPackage() string {
	return tf.pkg
}

func (tf *testFileWithAPI) GetExportedDecls() []ExportedDecl {
	return tf.decls
}

// Test adapter for ExportedDecl
type testExportedDecl struct {
	name       string
	kind       string
	signature  string
	properties []string
}

func (te *testExportedDecl) GetName() string {
	return te.name
}

func (te *testExportedDecl) GetKind() string {
	return te.kind
}

func (te *testExportedDecl) GetSignature() string {
	return te.signature
}

func (te *testExportedDecl) GetProperties() []string {
	return te.properties
}

func TestGenerateAPIMarkdown_Basic(t *testing.T) {
	files := []FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/service/service.go",
			pkg:     "service",
			decls: []ExportedDecl{
				&testExportedDecl{name: "Service", kind: "type", signature: "Service", properties: []string{"Name string", "Port int"}},
				&testExportedDecl{name: "NewService", kind: "func", signature: "NewService(string) *Service"},
				&testExportedDecl{name: "(*Service) Start", kind: "func", signature: "(*Service) Start() error"},
				&testExportedDecl{name: "MaxRetries", kind: "const", signature: "MaxRetries"},
				&testExportedDecl{name: "DefaultConfig", kind: "var", signature: "DefaultConfig"},
			},
		},
	}

	result := GenerateAPIMarkdown(files)

	// Check header
	if !strings.Contains(result, "# Public API") {
		t.Error("missing Public API header")
	}

	// Check package section
	if !strings.Contains(result, "## service") {
		t.Error("missing service package section")
	}

	// Check Types section
	if !strings.Contains(result, "### Types") {
		t.Error("missing Types section")
	}

	// Check type (bold because it has methods)
	if !strings.Contains(result, "- **Service**") {
		t.Error("missing Service type")
	}

	// Check properties under type
	if !strings.Contains(result, "  - Properties:") {
		t.Error("missing Properties subsection")
	}
	if !strings.Contains(result, "    - Name string") {
		t.Error("missing Name property")
	}

	// Check methods under type
	if !strings.Contains(result, "  - Methods:") {
		t.Error("missing Methods subsection")
	}
	if !strings.Contains(result, "    - (*Service) Start() error") {
		t.Error("missing Start method")
	}

	// Check package functions section
	if !strings.Contains(result, "### Package Functions") {
		t.Error("missing Package Functions section")
	}
	if !strings.Contains(result, "- NewService(string) *Service") {
		t.Error("missing NewService function")
	}

	// Check constants section
	if !strings.Contains(result, "### Constants") {
		t.Error("missing Constants section")
	}
	if !strings.Contains(result, "- MaxRetries") {
		t.Error("missing MaxRetries constant")
	}

	// Check variables section
	if !strings.Contains(result, "### Variables") {
		t.Error("missing Variables section")
	}
	if !strings.Contains(result, "- DefaultConfig") {
		t.Error("missing DefaultConfig variable")
	}
}

func TestGenerateAPIMarkdown_MultiplePackages(t *testing.T) {
	files := []FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/client/client.go",
			pkg:     "client",
			decls: []ExportedDecl{
				&testExportedDecl{name: "Client", kind: "type", signature: "Client"},
			},
		},
		&testFileWithAPI{
			relPath: "pkg/server/server.go",
			pkg:     "server",
			decls: []ExportedDecl{
				&testExportedDecl{name: "Server", kind: "type", signature: "Server"},
			},
		},
	}

	result := GenerateAPIMarkdown(files)

	// Check both packages are present
	if !strings.Contains(result, "## client") {
		t.Error("missing client package")
	}
	if !strings.Contains(result, "## server") {
		t.Error("missing server package")
	}

	// Check type names (italic because no methods)
	if !strings.Contains(result, "- *Client*") {
		t.Error("missing Client type")
	}
	if !strings.Contains(result, "- *Server*") {
		t.Error("missing Server type")
	}
}

func TestGenerateAPIMarkdown_NoExportedDeclarations(t *testing.T) {
	files := []FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/internal/helper.go",
			pkg:     "internal",
			decls:   []ExportedDecl{},
		},
	}

	result := GenerateAPIMarkdown(files)

	// Package with no exported declarations should not appear
	if strings.Contains(result, "## internal") {
		t.Error("package with no exported declarations should be skipped")
	}
}

func TestGenerateAPIMarkdown_SamePackageMultipleFiles(t *testing.T) {
	files := []FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/api/client.go",
			pkg:     "api",
			decls: []ExportedDecl{
				&testExportedDecl{name: "Client", kind: "type", signature: "Client"},
			},
		},
		&testFileWithAPI{
			relPath: "pkg/api/server.go",
			pkg:     "api",
			decls: []ExportedDecl{
				&testExportedDecl{name: "Server", kind: "type", signature: "Server"},
			},
		},
	}

	result := GenerateAPIMarkdown(files)

	// Should have one section for api package
	if !strings.Contains(result, "## api") {
		t.Error("missing api package")
	}

	// Should have both types (italic because no methods)
	if !strings.Contains(result, "- *Client*") {
		t.Error("missing Client type")
	}
	if !strings.Contains(result, "- *Server*") {
		t.Error("missing Server type")
	}
}

func TestGenerateAPIMarkdown_OnlyFunctions(t *testing.T) {
	files := []FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/utils/utils.go",
			pkg:     "utils",
			decls: []ExportedDecl{
				&testExportedDecl{name: "Format", kind: "func", signature: "Format(string) string"},
				&testExportedDecl{name: "Parse", kind: "func", signature: "Parse(string) error"},
			},
		},
	}

	result := GenerateAPIMarkdown(files)

	// Should have Package Functions section
	if !strings.Contains(result, "### Package Functions") {
		t.Error("missing Package Functions section")
	}

	// Should not have other sections
	if strings.Contains(result, "### Types") {
		t.Error("should not have Types section")
	}
	if strings.Contains(result, "### Constants") {
		t.Error("should not have Constants section")
	}
	if strings.Contains(result, "### Variables") {
		t.Error("should not have Variables section")
	}
}
