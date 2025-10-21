package validator

import (
	"fmt"
	"path/filepath"
	"strings"
)

// FileWithTestInfo interface for accessing file test-related information
type FileWithTestInfo interface {
	GetRelPath() string
	GetBaseName() string
	GetIsTest() bool
}

// validateTestNaming checks for strict 1:1 test file naming convention
// For each foo.go, there must be exactly one foo_test.go in the same directory
func (v *Validator) validateTestNaming() []Violation {
	if !v.cfg.ShouldEnforceStrictTestNaming() {
		return nil
	}

	var violations []Violation

	// Group files by directory and base name
	// map[directory]map[baseName]FileGroup
	fileGroups := make(map[string]map[string]*fileGroup)

	for _, node := range v.graph.GetNodes() {
		// We need to access BaseName and IsTest from the file info
		// This will be provided via adapter in pkg/linter
		fileInfo, ok := node.(FileWithTestInfo)
		if !ok {
			// Skip if the node doesn't provide test info
			continue
		}

		relPath := fileInfo.GetRelPath()
		baseName := fileInfo.GetBaseName()
		isTest := fileInfo.GetIsTest()
		dir := filepath.Dir(relPath)

		// Exclude special files that don't need tests
		if shouldExcludeFromTestNaming(baseName) {
			continue
		}

		// Initialize maps if needed
		if fileGroups[dir] == nil {
			fileGroups[dir] = make(map[string]*fileGroup)
		}
		if fileGroups[dir][baseName] == nil {
			fileGroups[dir][baseName] = &fileGroup{
				directory: dir,
				baseName:  baseName,
			}
		}

		group := fileGroups[dir][baseName]
		if isTest {
			group.testFiles = append(group.testFiles, relPath)
		} else {
			group.implFiles = append(group.implFiles, relPath)
		}
	}

	// Validate each file group
	for dir, groups := range fileGroups {
		for baseName, group := range groups {
			violations = append(violations, validateFileGroup(dir, baseName, group)...)
		}
	}

	return violations
}

// fileGroup represents all files with the same base name in a directory
type fileGroup struct {
	directory string
	baseName  string
	implFiles []string // Non-test files (e.g., foo.go)
	testFiles []string // Test files (e.g., foo_test.go)
}

// validateFileGroup validates that a file group follows strict 1:1 naming
func validateFileGroup(dir, baseName string, group *fileGroup) []Violation {
	var violations []Violation

	implCount := len(group.implFiles)
	testCount := len(group.testFiles)

	// Case 1: Multiple implementation files with same base name (shouldn't happen, but check)
	if implCount > 1 {
		for _, implFile := range group.implFiles {
			violations = append(violations, Violation{
				Type:  ViolationTestNaming,
				File:  implFile,
				Issue: fmt.Sprintf("Multiple implementation files found with base name '%s' in directory '%s'", baseName, dir),
				Rule:  "strict_test_naming: Each base name should have exactly one implementation file per directory",
				Fix:   fmt.Sprintf("Rename or consolidate duplicate implementation files with base name '%s'", baseName),
			})
		}
	}

	// Case 2: Multiple test files with same base name
	if testCount > 1 {
		for _, testFile := range group.testFiles {
			violations = append(violations, Violation{
				Type:  ViolationTestNaming,
				File:  testFile,
				Issue: fmt.Sprintf("Multiple test files found with base name '%s' in directory '%s'", baseName, dir),
				Rule:  "strict_test_naming: Each implementation file should have exactly one corresponding test file (foo.go -> foo_test.go)",
				Fix:   fmt.Sprintf("Consolidate test files into single '%s_test.go' file, or rename to use different base names", baseName),
			})
		}
	}

	// Case 3: Implementation file exists but no test file
	if implCount == 1 && testCount == 0 {
		violations = append(violations, Violation{
			Type:  ViolationTestNaming,
			File:  group.implFiles[0],
			Issue: fmt.Sprintf("Implementation file '%s' has no corresponding test file", filepath.Base(group.implFiles[0])),
			Rule:  "strict_test_naming: Each implementation file must have a corresponding test file (foo.go -> foo_test.go)",
			Fix:   fmt.Sprintf("Create test file '%s_test.go' in the same directory", baseName),
		})
	}

	// Case 4: Test file exists but no implementation file (orphaned test)
	if implCount == 0 && testCount == 1 {
		violations = append(violations, Violation{
			Type:  ViolationTestNaming,
			File:  group.testFiles[0],
			Issue: fmt.Sprintf("Test file '%s' has no corresponding implementation file", filepath.Base(group.testFiles[0])),
			Rule:  "strict_test_naming: Each test file must have a corresponding implementation file (foo_test.go -> foo.go)",
			Fix:   fmt.Sprintf("Create implementation file '%s.go' in the same directory, or remove/rename the orphaned test file", baseName),
		})
	}

	// Case 5: No implementation but multiple test files (compound violation)
	if implCount == 0 && testCount > 1 {
		for _, testFile := range group.testFiles {
			violations = append(violations, Violation{
				Type:  ViolationTestNaming,
				File:  testFile,
				Issue: fmt.Sprintf("Multiple orphaned test files with base name '%s' and no implementation file", baseName),
				Rule:  "strict_test_naming: Each test file must have a corresponding implementation file",
				Fix:   fmt.Sprintf("Create implementation file '%s.go' or consolidate/rename test files", baseName),
			})
		}
	}

	return violations
}

// shouldExcludeFromTestNaming determines if a file should be excluded from test naming validation
// These are special files that typically don't need corresponding test files
func shouldExcludeFromTestNaming(baseName string) bool {
	// Exclude documentation files
	if baseName == "doc" {
		return true
	}

	// Exclude common generated files (protobuf, code generation, etc.)
	excludeSuffixes := []string{
		"_gen",
		"_generated",
		".pb",
		"_mock",
		"_mocks",
	}
	for _, suffix := range excludeSuffixes {
		if strings.HasSuffix(baseName, suffix) {
			return true
		}
	}

	// Exclude test helper files (these are test files themselves)
	// Note: These will have IsTest=true, so they'll be in testFiles list
	// But we don't want to require foo.go for foo_helper_test.go
	if strings.Contains(baseName, "_helper") || strings.Contains(baseName, "testutil") {
		return true
	}

	return false
}
