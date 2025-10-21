package validator_test

import (
	"strings"
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/validator"
)

// Test config mock for strict test naming
type testNamingConfig struct {
	strictTestNaming bool
}

func (c *testNamingConfig) GetDirectoriesImport() map[string][]string {
	return nil
}

func (c *testNamingConfig) ShouldDetectUnused() bool {
	return false
}

func (c *testNamingConfig) GetRequiredDirectories() map[string]string {
	return nil
}

func (c *testNamingConfig) ShouldAllowOtherDirectories() bool {
	return true
}

func (c *testNamingConfig) ShouldDetectSharedExternalImports() bool {
	return false
}

func (c *testNamingConfig) GetSharedExternalImportsMode() string {
	return "warn"
}

func (c *testNamingConfig) GetSharedExternalImportsExclusions() []string {
	return nil
}

func (c *testNamingConfig) GetSharedExternalImportsExclusionPatterns() []string {
	return nil
}

func (c *testNamingConfig) ShouldLintTestFiles() bool {
	return false
}

func (c *testNamingConfig) GetTestExemptImports() []string {
	return nil
}

func (c *testNamingConfig) GetTestFileLocation() string {
	return "colocated"
}

func (c *testNamingConfig) ShouldRequireBlackboxTests() bool {
	return false
}

func (c *testNamingConfig) IsCoverageEnabled() bool {
	return false
}

func (c *testNamingConfig) GetCoverageThreshold() float64 {
	return 0
}

func (c *testNamingConfig) GetPackageThresholds() map[string]float64 {
	return nil
}

func (c *testNamingConfig) GetModule() string {
	return "test/module"
}

func (c *testNamingConfig) ShouldEnforceStrictTestNaming() bool {
	return c.strictTestNaming
}

// Mock file node with test info
type mockFileNodeWithTestInfo struct {
	relPath  string
	baseName string
	isTest   bool
}

func (m *mockFileNodeWithTestInfo) GetRelPath() string {
	return m.relPath
}

func (m *mockFileNodeWithTestInfo) GetPackage() string {
	return "testpkg"
}

func (m *mockFileNodeWithTestInfo) GetDependencies() []validator.Dependency {
	return nil
}

func (m *mockFileNodeWithTestInfo) GetBaseName() string {
	return m.baseName
}

func (m *mockFileNodeWithTestInfo) GetIsTest() bool {
	return m.isTest
}

// Mock graph
type mockGraphWithTestInfo struct {
	nodes []validator.FileNode
}

func (g *mockGraphWithTestInfo) GetNodes() []validator.FileNode {
	return g.nodes
}

func TestValidateTestNaming_Disabled(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: false}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo.go",
				baseName: "foo",
				isTest:   false,
			},
			// Missing test file
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have no violations when feature is disabled
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations when strict_test_naming is disabled, got %d", len(violations))
	}
}

func TestValidateTestNaming_Valid_OneToOne(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: true}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo.go",
				baseName: "foo",
				isTest:   false,
			},
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo_test.go",
				baseName: "foo",
				isTest:   true,
			},
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have no violations for valid 1:1 mapping
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for valid 1:1 mapping, got %d: %v", len(violations), violations)
	}
}

func TestValidateTestNaming_MissingTestFile(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: true}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo.go",
				baseName: "foo",
				isTest:   false,
			},
			// No test file
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have 1 violation for missing test file
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation for missing test file, got %d", len(violations))
	}

	violation := violations[0]
	if violation.GetType() != "Test Naming Convention" {
		t.Errorf("Expected violation type 'Test Naming Convention', got '%s'", violation.GetType())
	}
	if violation.GetFile() != "pkg/foo.go" {
		t.Errorf("Expected violation file 'pkg/foo.go', got '%s'", violation.GetFile())
	}
	if !strings.Contains(violation.GetIssue(), "no corresponding test file") {
		t.Errorf("Expected issue to mention missing test file, got: %s", violation.GetIssue())
	}
}

func TestValidateTestNaming_OrphanedTestFile(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: true}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo_test.go",
				baseName: "foo",
				isTest:   true,
			},
			// No implementation file
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have 1 violation for orphaned test file
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation for orphaned test file, got %d", len(violations))
	}

	violation := violations[0]
	if violation.GetType() != "Test Naming Convention" {
		t.Errorf("Expected violation type 'Test Naming Convention', got '%s'", violation.GetType())
	}
	if violation.GetFile() != "pkg/foo_test.go" {
		t.Errorf("Expected violation file 'pkg/foo_test.go', got '%s'", violation.GetFile())
	}
	if !strings.Contains(violation.GetIssue(), "no corresponding implementation file") {
		t.Errorf("Expected issue to mention missing implementation file, got: %s", violation.GetIssue())
	}
}

func TestValidateTestNaming_MultipleTestFiles(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: true}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo.go",
				baseName: "foo",
				isTest:   false,
			},
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo_test.go",
				baseName: "foo",
				isTest:   true,
			},
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo_integration_test.go", // Same base name!
				baseName: "foo",                         // Would need to be named differently
				isTest:   true,
			},
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have 2 violations - one for each test file
	if len(violations) != 2 {
		t.Fatalf("Expected 2 violations for multiple test files, got %d", len(violations))
	}

	for _, violation := range violations {
		if violation.GetType() != "Test Naming Convention" {
			t.Errorf("Expected violation type 'Test Naming Convention', got '%s'", violation.GetType())
		}
		if !strings.Contains(violation.GetIssue(), "Multiple test files") {
			t.Errorf("Expected issue to mention multiple test files, got: %s", violation.GetIssue())
		}
	}
}

func TestValidateTestNaming_DifferentDirectories(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: true}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			// Directory 1: valid 1:1
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo.go",
				baseName: "foo",
				isTest:   false,
			},
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo_test.go",
				baseName: "foo",
				isTest:   true,
			},
			// Directory 2: missing test
			&mockFileNodeWithTestInfo{
				relPath:  "internal/bar.go",
				baseName: "bar",
				isTest:   false,
			},
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have 1 violation only for internal/bar.go (missing test)
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	violation := violations[0]
	if violation.GetFile() != "internal/bar.go" {
		t.Errorf("Expected violation file 'internal/bar.go', got '%s'", violation.GetFile())
	}
}

func TestValidateTestNaming_ExcludeDocFiles(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: true}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/doc.go",
				baseName: "doc",
				isTest:   false,
			},
			// No test file for doc.go - should be excluded
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have 0 violations - doc.go is excluded
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for doc.go (should be excluded), got %d", len(violations))
	}
}

func TestValidateTestNaming_ExcludeGeneratedFiles(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: true}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/proto_gen.go",
				baseName: "proto_gen",
				isTest:   false,
			},
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/types.pb.go",
				baseName: "types.pb",
				isTest:   false,
			},
			// No test files for generated files - should be excluded
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have 0 violations - generated files are excluded
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for generated files (should be excluded), got %d", len(violations))
	}
}

func TestValidateTestNaming_MultipleDifferentBaseNames(t *testing.T) {
	cfg := &testNamingConfig{strictTestNaming: true}
	graph := &mockGraphWithTestInfo{
		nodes: []validator.FileNode{
			// foo: valid
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo.go",
				baseName: "foo",
				isTest:   false,
			},
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/foo_test.go",
				baseName: "foo",
				isTest:   true,
			},
			// bar: valid
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/bar.go",
				baseName: "bar",
				isTest:   false,
			},
			&mockFileNodeWithTestInfo{
				relPath:  "pkg/bar_test.go",
				baseName: "bar",
				isTest:   true,
			},
		},
	}

	v := validator.New(cfg, graph)
	violations := v.Validate()

	// Should have 0 violations - all have valid 1:1 mapping
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for multiple valid 1:1 mappings, got %d", len(violations))
	}
}
