package output_test

import (
	"strings"
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/output"
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
	dependencies []output.Dependency
}

func (tfn *testFileNode) GetRelPath() string           { return tfn.relPath }
func (tfn *testFileNode) GetPackage() string           { return tfn.pkg }
func (tfn *testFileNode) GetDependencies() []output.Dependency { return tfn.dependencies }

type testGraph struct {
	nodes []output.FileNode
}

func (tg *testGraph) GetNodes() []output.FileNode { return tg.nodes }

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
		nodes: []output.FileNode{
			&testFileNode{
				relPath: "cmd/api/main.go",
				pkg:     "main",
				dependencies: []output.Dependency{
					&testDependency{importPath: "github.com/test/project/pkg/service", isLocal: true, localPath: "pkg/service"},
				},
			},
			&testFileNode{
				relPath: "pkg/service/service.go",
				pkg:     "service",
				dependencies: []output.Dependency{
					&testDependency{importPath: "github.com/external/lib", isLocal: false},
				},
			},
		},
	}

	md := output.GenerateMarkdown(g)

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
		nodes: []output.FileNode{
			&testFileNode{
				relPath:      "pkg/types/types.go",
				pkg:          "types",
				dependencies: []output.Dependency{},
			},
		},
	}

	md := output.GenerateMarkdown(g)

	if !strings.Contains(md, "depends on: (none)") {
		t.Error("expected 'depends on: (none)' for file with no dependencies")
	}
}

func TestFormatViolations_NoViolations(t *testing.T) {
	var violations []output.Violation
	result := output.FormatViolations(violations)

	if result != "" {
		t.Error("expected empty string for no violations")
	}
}

func TestFormatViolations_WithViolations(t *testing.T) {
	violations := []output.Violation{
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

	result := output.FormatViolations(violations)

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
	violations := []output.Violation{
		&testViolation{
			violationType: "Unused Package",
			issue:         "Package pkg/legacy not imported by any cmd/ package",
			rule:          "All packages should be transitively imported from cmd/",
			fix:           "Remove package or add import from cmd/",
		},
	}

	result := output.FormatViolations(violations)

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
	decls   []output.ExportedDecl
}

func (tf *testFileWithAPI) GetRelPath() string {
	return tf.relPath
}

func (tf *testFileWithAPI) GetPackage() string {
	return tf.pkg
}

func (tf *testFileWithAPI) GetExportedDecls() []output.ExportedDecl {
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
	files := []output.FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/service/service.go",
			pkg:     "service",
			decls: []output.ExportedDecl{
				&testExportedDecl{name: "Service", kind: "type", signature: "Service", properties: []string{"Name string", "Port int"}},
				&testExportedDecl{name: "NewService", kind: "func", signature: "NewService(string) *Service"},
				&testExportedDecl{name: "(*Service) Start", kind: "func", signature: "(*Service) Start() error"},
				&testExportedDecl{name: "MaxRetries", kind: "const", signature: "MaxRetries"},
				&testExportedDecl{name: "DefaultConfig", kind: "var", signature: "DefaultConfig"},
			},
		},
	}

	result := output.GenerateAPIMarkdown(files)

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
	files := []output.FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/client/client.go",
			pkg:     "client",
			decls: []output.ExportedDecl{
				&testExportedDecl{name: "Client", kind: "type", signature: "Client"},
			},
		},
		&testFileWithAPI{
			relPath: "pkg/server/server.go",
			pkg:     "server",
			decls: []output.ExportedDecl{
				&testExportedDecl{name: "Server", kind: "type", signature: "Server"},
			},
		},
	}

	result := output.GenerateAPIMarkdown(files)

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
	files := []output.FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/internal/helper.go",
			pkg:     "internal",
			decls:   []output.ExportedDecl{},
		},
	}

	result := output.GenerateAPIMarkdown(files)

	// Package with no exported declarations should not appear
	if strings.Contains(result, "## internal") {
		t.Error("package with no exported declarations should be skipped")
	}
}

func TestGenerateAPIMarkdown_SamePackageMultipleFiles(t *testing.T) {
	files := []output.FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/api/client.go",
			pkg:     "api",
			decls: []output.ExportedDecl{
				&testExportedDecl{name: "Client", kind: "type", signature: "Client"},
			},
		},
		&testFileWithAPI{
			relPath: "pkg/api/server.go",
			pkg:     "api",
			decls: []output.ExportedDecl{
				&testExportedDecl{name: "Server", kind: "type", signature: "Server"},
			},
		},
	}

	result := output.GenerateAPIMarkdown(files)

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
	files := []output.FileWithAPI{
		&testFileWithAPI{
			relPath: "pkg/utils/utils.go",
			pkg:     "utils",
			decls: []output.ExportedDecl{
				&testExportedDecl{name: "Format", kind: "func", signature: "Format(string) string"},
				&testExportedDecl{name: "Parse", kind: "func", signature: "Parse(string) error"},
			},
		},
	}

	result := output.GenerateAPIMarkdown(files)

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

func TestGenerateFullDocumentation_Complete(t *testing.T) {
	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{
				"cmd":      "Command-line entry points",
				"pkg":      "Public API",
				"internal": "Domain primitives",
			},
			AllowOtherDirectories: true,
			ExistingDirs: map[string]bool{
				"cmd":      true,
				"pkg":      true,
				"internal": true,
			},
		},
		Rules: output.RulesInfo{
			DirectoriesImport: map[string][]string{
				"cmd":      {"pkg"},
				"pkg":      {"internal"},
				"internal": {},
			},
			DetectUnused: true,
		},
		Graph: &testGraph{
			nodes: []output.FileNode{
				&testFileNode{
					relPath: "cmd/main.go",
					pkg:     "main",
					dependencies: []output.Dependency{
						&testDependency{importPath: "github.com/test/pkg/service", isLocal: true, localPath: "pkg/service"},
					},
				},
			},
		},
		Files: []output.FileWithAPI{
			&testFileWithAPI{
				relPath: "pkg/service/service.go",
				pkg:     "service",
				decls: []output.ExportedDecl{
					&testExportedDecl{name: "Service", kind: "type", signature: "Service"},
				},
			},
		},
		Violations:     []output.Violation{},
		ViolationCount: 0,
		FileCount:      5,
		PackageCount:   3,
	}

	result := output.GenerateFullDocumentation(doc)

	// Check main header
	if !strings.Contains(result, "# Project Architecture") {
		t.Error("missing main header")
	}

	// Check table of contents
	if !strings.Contains(result, "## Table of Contents") {
		t.Error("missing table of contents")
	}

	// Check structure section
	if !strings.Contains(result, "## Project Structure") {
		t.Error("missing structure section")
	}
	if !strings.Contains(result, "### cmd") {
		t.Error("missing cmd directory section")
	}
	if !strings.Contains(result, "**Status**: ✓ Exists") {
		t.Error("missing existence status")
	}

	// Check rules section
	if !strings.Contains(result, "## Architectural Rules") {
		t.Error("missing rules section")
	}
	if !strings.Contains(result, "**Validation Status**: ✓ No violations found") {
		t.Error("missing validation status")
	}

	// Check dependency graph section
	if !strings.Contains(result, "## Dependency Graph") {
		t.Error("missing dependency graph section")
	}

	// Check public API section
	if !strings.Contains(result, "## Public API") {
		t.Error("missing public API section")
	}

	// Check statistics section
	if !strings.Contains(result, "## Statistics") {
		t.Error("missing statistics section")
	}
	if !strings.Contains(result, "- **Total Files**: 5") {
		t.Error("missing file count")
	}
	if !strings.Contains(result, "- **Total Packages**: 3") {
		t.Error("missing package count")
	}
}

func TestGenerateFullDocumentation_WithViolations(t *testing.T) {
	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories:   map[string]string{"cmd": "Entry points"},
			AllowOtherDirectories: false,
			ExistingDirs:          map[string]bool{"cmd": false},
		},
		Rules: output.RulesInfo{
			DirectoriesImport: map[string][]string{"cmd": {"pkg"}},
			DetectUnused:      false,
		},
		Graph:          &testGraph{nodes: []output.FileNode{}},
		Files:          []output.FileWithAPI{},
		Violations:     []output.Violation{&testViolation{violationType: "Test"}},
		ViolationCount: 1,
		FileCount:      1,
		PackageCount:   1,
	}

	result := output.GenerateFullDocumentation(doc)

	// Check violation status
	if !strings.Contains(result, "**Validation Status**: ✗ 1 violation(s) found") {
		t.Error("missing violation status")
	}

	// Check strict mode
	if !strings.Contains(result, "**Strict Mode**: Only required directories are allowed") {
		t.Error("missing strict mode indicator")
	}

	// Check missing directory
	if !strings.Contains(result, "**Status**: ✗ Missing") {
		t.Error("missing directory status not shown")
	}
}

func TestGenerateFullDocumentation_NoRequiredDirs(t *testing.T) {
	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories:   map[string]string{},
			AllowOtherDirectories: true,
			ExistingDirs:          map[string]bool{},
		},
		Rules: output.RulesInfo{
			DirectoriesImport: map[string][]string{},
			DetectUnused:      false,
		},
		Graph:          &testGraph{nodes: []output.FileNode{}},
		Files:          []output.FileWithAPI{},
		Violations:     []output.Violation{},
		ViolationCount: 0,
		FileCount:      0,
		PackageCount:   0,
	}

	result := output.GenerateFullDocumentation(doc)

	if !strings.Contains(result, "No required directory structure defined") {
		t.Error("missing 'no required directory' message")
	}
}

func TestGenerateFullDocumentation_NoPublicAPI(t *testing.T) {
	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories:   map[string]string{"cmd": "Entry"},
			AllowOtherDirectories: true,
			ExistingDirs:          map[string]bool{"cmd": true},
		},
		Rules: output.RulesInfo{
			DirectoriesImport: map[string][]string{},
			DetectUnused:      false,
		},
		Graph:          &testGraph{nodes: []output.FileNode{}},
		Files:          []output.FileWithAPI{},
		Violations:     []output.Violation{},
		ViolationCount: 0,
		FileCount:      1,
		PackageCount:   1,
	}

	result := output.GenerateFullDocumentation(doc)

	if !strings.Contains(result, "No public API exported.") {
		t.Error("missing 'no public API' message")
	}
}

func TestFormatViolationsWithContext_Enabled(t *testing.T) {
	violations := []output.Violation{
		&testViolation{
			violationType: "Forbidden pkg-to-pkg Dependency",
			file:          "pkg/a/a.go",
			line:          5,
			issue:         "pkg/a imports pkg/b",
			rule:          "pkg packages must not import other pkg packages",
			fix:           "Use internal/ or interface",
		},
	}

	errorContext := &output.ErrorContext{
		Enabled:              true,
		PresetName:           "ddd",
		ArchitecturalGoals:   "Domain-Driven Design goals here",
		Principles:           []string{"Bounded contexts", "Explicit boundaries"},
		RefactoringGuidance:  "Refactor to use aggregates",
	}

	result := output.FormatViolationsWithContext(violations, errorContext)

	// Check architectural context header
	if !strings.Contains(result, "ARCHITECTURAL VIOLATIONS DETECTED") {
		t.Error("missing architectural violations header")
	}

	// Check preset name
	if !strings.Contains(result, "This project uses the 'ddd' architectural preset") {
		t.Error("missing preset name")
	}

	// Check architectural goals section
	if !strings.Contains(result, "ARCHITECTURAL GOALS") {
		t.Error("missing architectural goals section")
	}
	if !strings.Contains(result, "Domain-Driven Design goals here") {
		t.Error("missing architectural goals content")
	}

	// Check principles section
	if !strings.Contains(result, "KEY PRINCIPLES") {
		t.Error("missing principles section")
	}
	if !strings.Contains(result, "Bounded contexts") {
		t.Error("missing first principle")
	}
	if !strings.Contains(result, "Explicit boundaries") {
		t.Error("missing second principle")
	}

	// Check violations section
	if !strings.Contains(result, "VIOLATIONS") {
		t.Error("missing violations section")
	}

	// Check refactoring guidance (should appear for architectural violations)
	if !strings.Contains(result, "REFACTORING GUIDANCE") {
		t.Error("missing refactoring guidance section")
	}
	if !strings.Contains(result, "Refactor to use aggregates") {
		t.Error("missing refactoring guidance content")
	}

	// Check tip
	if !strings.Contains(result, "💡 TIP:") {
		t.Error("missing tip section")
	}
}

func TestFormatViolationsWithContext_Disabled(t *testing.T) {
	violations := []output.Violation{
		&testViolation{
			violationType: "Cross-cmd Dependency",
			file:          "cmd/a/main.go",
			issue:         "cmd/a imports cmd/b",
			rule:          "cmd packages must not import other cmd packages",
			fix:           "Extract to pkg/",
		},
	}

	errorContext := &output.ErrorContext{
		Enabled: false,
	}

	result := output.FormatViolationsWithContext(violations, errorContext)

	// Should use simple header
	if !strings.Contains(result, "DEPENDENCY VIOLATIONS DETECTED") {
		t.Error("missing simple violations header")
	}

	// Should NOT have architectural context
	if strings.Contains(result, "ARCHITECTURAL GOALS") {
		t.Error("should not have architectural goals when disabled")
	}
	if strings.Contains(result, "KEY PRINCIPLES") {
		t.Error("should not have principles when disabled")
	}
	if strings.Contains(result, "REFACTORING GUIDANCE") {
		t.Error("should not have refactoring guidance when disabled")
	}
}

func TestFormatViolationsWithContext_TestViolations(t *testing.T) {
	violations := []output.Violation{
		&testViolation{
			violationType: "Insufficient Test Coverage",
			file:          "pkg/service/service.go",
			issue:         "Coverage is 45.2%, needs 70%",
			rule:          "All packages must meet minimum coverage",
			fix:           "Add more tests",
		},
	}

	errorContext := &output.ErrorContext{
		Enabled:          true,
		CoverageGuidance: "Write tests for critical paths first",
	}

	result := output.FormatViolationsWithContext(violations, errorContext)

	// Check test coverage guidance appears
	if !strings.Contains(result, "TEST COVERAGE GUIDANCE") {
		t.Error("missing test coverage guidance section")
	}
	if !strings.Contains(result, "Write tests for critical paths first") {
		t.Error("missing coverage guidance content")
	}

	// Should NOT show refactoring guidance (only for architectural violations)
	if strings.Contains(result, "REFACTORING GUIDANCE") {
		t.Error("should not show refactoring guidance for test violations")
	}

	// Check appropriate tip
	if !strings.Contains(result, "Test coverage ensures your code works correctly") {
		t.Error("missing test coverage tip")
	}
}

func TestFormatViolationsWithContext_WhiteboxTestViolations(t *testing.T) {
	violations := []output.Violation{
		&testViolation{
			violationType: "Whitebox Test",
			file:          "pkg/service/service_test.go",
			issue:         "Test uses whitebox testing (package service)",
			rule:          "Tests should use blackbox testing",
			fix:           "Change to package service_test",
		},
	}

	errorContext := &output.ErrorContext{
		Enabled:                 true,
		BlackboxTestingGuidance: "Blackbox tests are more resilient",
	}

	result := output.FormatViolationsWithContext(violations, errorContext)

	// Check blackbox testing guidance appears
	if !strings.Contains(result, "BLACKBOX TESTING GUIDANCE") {
		t.Error("missing blackbox testing guidance section")
	}
	if !strings.Contains(result, "Blackbox tests are more resilient") {
		t.Error("missing blackbox testing guidance content")
	}

	// Should NOT show refactoring guidance or coverage guidance
	if strings.Contains(result, "REFACTORING GUIDANCE") {
		t.Error("should not show refactoring guidance for whitebox test violations")
	}
	if strings.Contains(result, "TEST COVERAGE GUIDANCE") {
		t.Error("should not show coverage guidance for whitebox test violations")
	}

	// Check appropriate tip
	if !strings.Contains(result, "Blackbox testing improves test resilience") {
		t.Error("missing blackbox testing tip")
	}
}

func TestFormatViolationsWithContext_MixedViolations(t *testing.T) {
	violations := []output.Violation{
		&testViolation{
			violationType: "Forbidden pkg-to-pkg Dependency",
			file:          "pkg/a/a.go",
			issue:         "pkg/a imports pkg/b",
			rule:          "No cross-pkg imports",
			fix:           "Use internal/",
		},
		&testViolation{
			violationType: "Insufficient Test Coverage",
			file:          "pkg/a/a.go",
			issue:         "Coverage is 30%",
			rule:          "Need 70%",
			fix:           "Add tests",
		},
	}

	errorContext := &output.ErrorContext{
		Enabled:              true,
		RefactoringGuidance:  "Refactoring steps",
		CoverageGuidance:     "Coverage steps",
	}

	result := output.FormatViolationsWithContext(violations, errorContext)

	// Both guidance sections should appear
	if !strings.Contains(result, "REFACTORING GUIDANCE") {
		t.Error("missing refactoring guidance for mixed violations")
	}
	if !strings.Contains(result, "TEST COVERAGE GUIDANCE") {
		t.Error("missing coverage guidance for mixed violations")
	}

	// Check appropriate tip for mixed violations
	if !strings.Contains(result, "Address architectural violations first, then improve test quality") {
		t.Error("missing mixed violations tip")
	}
}
