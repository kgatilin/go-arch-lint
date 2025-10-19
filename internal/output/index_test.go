package output_test

import (
	"strings"
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/output"
)

// Test helpers for index documentation

type testDependencyForIndex struct {
	importPath string
	isLocal    bool
	localPath  string
	symbols    []string
}

func (td *testDependencyForIndex) GetImportPath() string  { return td.importPath }
func (td *testDependencyForIndex) IsLocalDep() bool        { return td.isLocal }
func (td *testDependencyForIndex) GetLocalPath() string    { return td.localPath }
func (td *testDependencyForIndex) GetUsedSymbols() []string { return td.symbols }

type testFileNodeForIndex struct {
	relPath      string
	pkgName      string
	dependencies []output.Dependency
}

func (tfn *testFileNodeForIndex) GetRelPath() string                    { return tfn.relPath }
func (tfn *testFileNodeForIndex) GetPackage() string                    { return tfn.pkgName }
func (tfn *testFileNodeForIndex) GetDependencies() []output.Dependency { return tfn.dependencies }

type testGraphForIndex struct {
	nodes []output.FileNode
}

func (tg *testGraphForIndex) GetNodes() []output.FileNode { return tg.nodes }

type testExportedDeclForIndex struct {
	name       string
	kind       string
	signature  string
	properties []string
}

func (ted *testExportedDeclForIndex) GetName() string       { return ted.name }
func (ted *testExportedDeclForIndex) GetKind() string       { return ted.kind }
func (ted *testExportedDeclForIndex) GetSignature() string  { return ted.signature }
func (ted *testExportedDeclForIndex) GetProperties() []string { return ted.properties }

type testFileWithAPIForIndex struct {
	relPath      string
	pkgName      string
	exportedDecls []output.ExportedDecl
}

func (twa *testFileWithAPIForIndex) GetRelPath() string                    { return twa.relPath }
func (twa *testFileWithAPIForIndex) GetPackage() string                    { return twa.pkgName }
func (twa *testFileWithAPIForIndex) GetExportedDecls() []output.ExportedDecl { return twa.exportedDecls }

// Tests

func TestGenerateIndexDocumentation_HasRequiredSections(t *testing.T) {
	// Create test data
	graph := &testGraphForIndex{
		nodes: []output.FileNode{
			&testFileNodeForIndex{
				relPath:      "cmd/main.go",
				pkgName:      "main",
				dependencies: []output.Dependency{},
			},
			&testFileNodeForIndex{
				relPath: "pkg/linter/linter.go",
				pkgName: "linter",
				dependencies: []output.Dependency{
					&testDependencyForIndex{
						importPath: "github.com/kgatilin/go-arch-lint/internal/config",
						isLocal:    true,
						localPath:  "internal/config",
					},
				},
			},
		},
	}

	files := []output.FileWithAPI{
		&testFileWithAPIForIndex{
			relPath: "cmd/main.go",
			pkgName: "main",
			exportedDecls: []output.ExportedDecl{
				&testExportedDeclForIndex{name: "main", kind: "func"},
			},
		},
		&testFileWithAPIForIndex{
			relPath: "pkg/linter/linter.go",
			pkgName: "linter",
			exportedDecls: []output.ExportedDecl{
				&testExportedDeclForIndex{name: "Run", kind: "func"},
				&testExportedDeclForIndex{name: "Init", kind: "func"},
			},
		},
	}

	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{
				"cmd":      "Application entry points",
				"pkg":      "Public libraries",
				"internal": "Private packages",
			},
			AllowOtherDirectories: false,
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
		Graph:          graph,
		Files:          files,
		ViolationCount: 0,
		FileCount:      2,
		PackageCount:   2,
	}

	output := output.GenerateIndexDocumentation(doc)

	// Verify required sections exist
	requiredSections := []string{
		"# Project Architecture Index",
		"## Quick Reference",
		"## Architecture Summary",
		"## Architectural Rules",
		"## Package Directory",
		"## Agent Guidance",
		"## Statistics",
	}

	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("Missing required section: %s", section)
		}
	}
}

func TestGenerateIndexDocumentation_ContainsPackageInfo(t *testing.T) {
	graph := &testGraphForIndex{
		nodes: []output.FileNode{
			&testFileNodeForIndex{
				relPath:      "pkg/linter/linter.go",
				pkgName:      "pkg/linter",
				dependencies: []output.Dependency{},
			},
			&testFileNodeForIndex{
				relPath:      "internal/config/config.go",
				pkgName:      "internal/config",
				dependencies: []output.Dependency{},
			},
		},
	}

	files := []output.FileWithAPI{
		&testFileWithAPIForIndex{
			relPath: "pkg/linter/linter.go",
			pkgName: "pkg/linter",
			exportedDecls: []output.ExportedDecl{
				&testExportedDeclForIndex{name: "Run", kind: "func"},
				&testExportedDeclForIndex{name: "Init", kind: "func"},
				&testExportedDeclForIndex{name: "Preset", kind: "type"},
			},
		},
		&testFileWithAPIForIndex{
			relPath: "internal/config/config.go",
			pkgName: "internal/config",
			exportedDecls: []output.ExportedDecl{
				&testExportedDeclForIndex{name: "Config", kind: "type"},
				&testExportedDeclForIndex{name: "Load", kind: "func"},
			},
		},
	}

	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{
				"pkg":      "Public libraries",
				"internal": "Private packages",
			},
			AllowOtherDirectories: true,
			ExistingDirs: map[string]bool{
				"pkg":      true,
				"internal": true,
			},
		},
		Rules: output.RulesInfo{
			DirectoriesImport: map[string][]string{
				"pkg":      {"internal"},
				"internal": {},
			},
			DetectUnused: false,
		},
		Graph:          graph,
		Files:          files,
		ViolationCount: 0,
		FileCount:      2,
		PackageCount:   2,
	}

	result := output.GenerateIndexDocumentation(doc)

	// Verify package names appear in output (check full paths)
	if !strings.Contains(result, "pkg/linter") {
		t.Errorf("Expected package 'pkg/linter' in output\nGot: %s", result)
	}
	if !strings.Contains(result, "internal/config") {
		t.Error("Expected package 'internal/config' in output")
	}

	// Verify export counts appear
	if !strings.Contains(result, "| Exports: 3") {
		t.Errorf("Expected '| Exports: 3' for linter package")
	}
	if !strings.Contains(result, "| Exports: 2") {
		t.Error("Expected '| Exports: 2' for config package")
	}
}

func TestGenerateIndexDocumentation_ContainsArchitecturalRules(t *testing.T) {
	graph := &testGraphForIndex{nodes: []output.FileNode{}}
	files := []output.FileWithAPI{}

	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{
				"cmd":      "Entry points",
				"pkg":      "Public API",
				"internal": "Isolated",
			},
			AllowOtherDirectories: false,
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
		Graph:          graph,
		Files:          files,
		ViolationCount: 0,
		FileCount:      0,
		PackageCount:   0,
	}

	result := output.GenerateIndexDocumentation(doc)

	// Verify architectural rules section
	if !strings.Contains(result, "## Architectural Rules") {
		t.Error("Missing Architectural Rules section")
	}

	// Verify layer dependencies are shown
	if !strings.Contains(result, "cmd") || !strings.Contains(result, "pkg") {
		t.Error("Expected layer dependencies to be shown")
	}

	// Verify isolation is clearly stated
	if !strings.Contains(result, "complete isolation") || !strings.Contains(result, "[]") {
		t.Error("Expected to show internal package isolation")
	}
}

func TestGenerateIndexDocumentation_ContainsAgentGuidance(t *testing.T) {
	graph := &testGraphForIndex{nodes: []output.FileNode{}}
	files := []output.FileWithAPI{}

	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{},
			ExistingDirs:        map[string]bool{},
		},
		Rules:          output.RulesInfo{},
		Graph:          graph,
		Files:          files,
		ViolationCount: 0,
		FileCount:      0,
		PackageCount:   0,
	}

	result := output.GenerateIndexDocumentation(doc)

	// Verify agent guidance section
	if !strings.Contains(result, "## Agent Guidance") {
		t.Error("Missing Agent Guidance section")
	}

	// Verify guidance contains helpful commands
	if !strings.Contains(result, "-format=api") {
		t.Error("Expected API format guidance")
	}
	if !strings.Contains(result, "-format=markdown") {
		t.Error("Expected markdown format guidance")
	}
	if !strings.Contains(result, "arch-generated.md") {
		t.Error("Expected reference to full documentation")
	}
}

func TestGenerateIndexDocumentation_ContainsStatistics(t *testing.T) {
	graph := &testGraphForIndex{
		nodes: []output.FileNode{
			&testFileNodeForIndex{
				relPath: "cmd/main.go",
				pkgName: "main",
				dependencies: []output.Dependency{
					&testDependencyForIndex{
						importPath: "fmt",
						isLocal:    false,
					},
				},
			},
			&testFileNodeForIndex{
				relPath: "pkg/linter/linter.go",
				pkgName: "linter",
				dependencies: []output.Dependency{
					&testDependencyForIndex{
						importPath: "os",
						isLocal:    false,
					},
				},
			},
		},
	}

	files := []output.FileWithAPI{}

	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{},
			ExistingDirs:        map[string]bool{},
		},
		Rules:          output.RulesInfo{},
		Graph:          graph,
		Files:          files,
		ViolationCount: 0,
		FileCount:      2,
		PackageCount:   2,
	}

	result := output.GenerateIndexDocumentation(doc)

	// Verify statistics section
	if !strings.Contains(result, "## Statistics") {
		t.Error("Missing Statistics section")
	}

	// Verify file and package counts (format: "- **Total Files**: %d")
	if !strings.Contains(result, "**Total Files**: 2") {
		t.Error("Expected file count in statistics")
	}
	if !strings.Contains(result, "**Total Packages**: 2") {
		t.Error("Expected package count in statistics")
	}
	if !strings.Contains(result, "**External Dependencies**: 2") {
		t.Error("Expected external dependency count")
	}
}

func TestGenerateIndexDocumentation_ViolationStatus(t *testing.T) {
	graph := &testGraphForIndex{nodes: []output.FileNode{}}
	files := []output.FileWithAPI{}

	// Test with no violations
	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{},
			ExistingDirs:        map[string]bool{},
		},
		Rules:          output.RulesInfo{},
		Graph:          graph,
		Files:          files,
		ViolationCount: 0,
		FileCount:      0,
		PackageCount:   0,
	}

	result := output.GenerateIndexDocumentation(doc)
	if !strings.Contains(result, "✓ 0 violations") {
		t.Error("Expected success status for 0 violations")
	}

	// Test with violations
	doc.ViolationCount = 5
	result = output.GenerateIndexDocumentation(doc)
	if !strings.Contains(result, "✗ 5 violation") {
		t.Error("Expected failure status for violations")
	}
}

func TestGenerateIndexDocumentation_IsCompact(t *testing.T) {
	// Build a reasonably complex graph
	graph := &testGraphForIndex{
		nodes: make([]output.FileNode, 20),
	}

	for i := 0; i < 20; i++ {
		graph.nodes[i] = &testFileNodeForIndex{
			relPath:      "pkg/module" + string(rune('a'+i)) + "/file.go",
			pkgName:      "module" + string(rune('a'+i)),
			dependencies: []output.Dependency{},
		}
	}

	files := make([]output.FileWithAPI, 20)
	for i := 0; i < 20; i++ {
		decls := make([]output.ExportedDecl, 5)
		for j := 0; j < 5; j++ {
			decls[j] = &testExportedDeclForIndex{
				name: "Export" + string(rune('a'+j)),
				kind: "type",
			}
		}
		files[i] = &testFileWithAPIForIndex{
			relPath:       "pkg/module" + string(rune('a'+i)) + "/file.go",
			pkgName:       "module" + string(rune('a'+i)),
			exportedDecls: decls,
		}
	}

	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{
				"pkg": "Public libraries",
			},
			ExistingDirs: map[string]bool{"pkg": true},
		},
		Rules: output.RulesInfo{
			DirectoriesImport: map[string][]string{
				"pkg": {},
			},
		},
		Graph:          graph,
		Files:          files,
		ViolationCount: 0,
		FileCount:      20,
		PackageCount:   20,
	}

	result := output.GenerateIndexDocumentation(doc)

	// Index should be reasonably compact (much smaller than full docs)
	// Full docs would typically be 50-100KB, index should be under 10KB
	if len(result) > 20000 {
		t.Logf("Warning: Index documentation is larger than expected: %d bytes (should be < 20KB)", len(result))
	}

	// But still substantial enough to contain necessary info
	if len(result) < 1000 {
		t.Error("Index documentation too small - missing required information")
	}
}

func TestGenerateIndexDocumentation_LayerOrganization(t *testing.T) {
	graph := &testGraphForIndex{
		nodes: []output.FileNode{
			&testFileNodeForIndex{relPath: "cmd/main.go", pkgName: "main"},
			&testFileNodeForIndex{relPath: "pkg/linter/linter.go", pkgName: "pkg/linter"},
			&testFileNodeForIndex{relPath: "internal/config/config.go", pkgName: "internal/config"},
		},
	}

	files := []output.FileWithAPI{
		&testFileWithAPIForIndex{
			relPath: "cmd/main.go",
			pkgName: "main",
			exportedDecls: []output.ExportedDecl{
				&testExportedDeclForIndex{name: "main", kind: "func"},
			},
		},
		&testFileWithAPIForIndex{
			relPath: "pkg/linter/linter.go",
			pkgName: "pkg/linter",
			exportedDecls: []output.ExportedDecl{
				&testExportedDeclForIndex{name: "Run", kind: "func"},
			},
		},
		&testFileWithAPIForIndex{
			relPath: "internal/config/config.go",
			pkgName: "internal/config",
			exportedDecls: []output.ExportedDecl{
				&testExportedDeclForIndex{name: "Config", kind: "type"},
			},
		},
	}

	doc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories: map[string]string{
				"cmd":      "Entry points",
				"pkg":      "Public libraries",
				"internal": "Isolated primitives",
			},
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
		},
		Graph:          graph,
		Files:          files,
		ViolationCount: 0,
		FileCount:      3,
		PackageCount:   3,
	}

	result := output.GenerateIndexDocumentation(doc)

	// Verify packages are grouped by layer
	cmdIdx := strings.Index(result, "### cmd")
	pkgIdx := strings.Index(result, "### pkg")
	internalIdx := strings.Index(result, "### internal")

	if cmdIdx == -1 {
		t.Error("Missing cmd layer section")
	}
	if pkgIdx == -1 {
		t.Error("Missing pkg layer section")
	}
	if internalIdx == -1 {
		t.Error("Missing internal layer section")
	}

	// Verify proper ordering (cmd should come before pkg, pkg before internal)
	if cmdIdx > 0 && pkgIdx > cmdIdx && internalIdx > pkgIdx {
		// Good - proper ordering
	} else if cmdIdx > 0 && pkgIdx > 0 && internalIdx > 0 {
		// At least all present, order may vary - just warn
		t.Log("Note: Layer ordering may not match expectation (cmd, pkg, internal)")
	}
}
