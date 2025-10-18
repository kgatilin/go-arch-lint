package output

import (
	"strings"
	"testing"
)

// Test structs that implement output interfaces for testing

type testDependency struct {
	importPath  string
	isLocal     bool
	localPath   string
	usedSymbols []string
}

func (td *testDependency) GetImportPath() string   { return td.importPath }
func (td *testDependency) IsLocalDep() bool        { return td.isLocal }
func (td *testDependency) GetLocalPath() string    { return td.localPath }
func (td *testDependency) GetUsedSymbols() []string { return td.usedSymbols }

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

type testViolation struct {
	violationType string
	file          string
	line          int
	issue         string
	rule          string
	fix           string
}

func (tv *testViolation) GetType() string  { return tv.violationType }
func (tv *testViolation) GetFile() string  { return tv.file }
func (tv *testViolation) GetLine() int     { return tv.line }
func (tv *testViolation) GetIssue() string { return tv.issue }
func (tv *testViolation) GetRule() string  { return tv.rule }
func (tv *testViolation) GetFix() string   { return tv.fix }

func TestGenerateMarkdown_Basic(t *testing.T) {
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath: "cmd/api/main.go",
				pkg:     "main",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/test/project/pkg/service", isLocal: true, localPath: "pkg/service"},
				},
			},
			&testFileNode{
				relPath: "pkg/service/service.go",
				pkg:     "service",
				dependencies: []Dependency{
					&testDependency{importPath: "github.com/external/lib", isLocal: false},
				},
			},
		},
	}

	md := GenerateMarkdown(g)

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
	g := &testGraph{
		nodes: []FileNode{
			&testFileNode{
				relPath:      "pkg/types/types.go",
				pkg:          "types",
				dependencies: []Dependency{},
			},
		},
	}

	md := GenerateMarkdown(g)

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
	violations := []Violation{
		&testViolation{
			violationType: "Forbidden pkg-to-pkg Dependency",
			file:          "pkg/http/handler.go",
			issue:         "pkg/http imports pkg/database",
			rule:          "pkg packages must not import other pkg packages",
			fix:           "Import from internal/ or define interface locally",
		},
		&testViolation{
			violationType: "Cross-cmd Dependency",
			file:          "cmd/api/main.go",
			line:          10,
			issue:         "cmd/api imports cmd/worker",
			rule:          "cmd packages must not import other cmd packages",
			fix:           "Extract shared code to pkg/ or internal/",
		},
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
	violations := []Violation{
		&testViolation{
			violationType: "Unused Package",
			issue:         "Package pkg/legacy not imported by any cmd/ package",
			rule:          "All packages should be transitively imported from cmd/",
			fix:           "Remove package or add import from cmd/",
		},
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
