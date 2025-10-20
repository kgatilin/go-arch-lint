package validator

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateTestFileLocations checks that test files are in the correct location based on policy
func (v *Validator) validateTestFileLocations() []Violation {
	var violations []Violation

	policy := v.cfg.GetTestFileLocation()

	for _, node := range v.graph.GetNodes() {
		relPath := node.GetRelPath()

		// Only check test files
		if !strings.HasSuffix(relPath, "_test.go") {
			continue
		}

		switch policy {
		case "colocated":
			// Test files should be next to the code they're testing (not in a separate tests/ directory)
			if strings.HasPrefix(relPath, "tests/") || strings.Contains(relPath, "/tests/") {
				violations = append(violations, Violation{
					Type:  ViolationTestFileLocation,
					File:  relPath,
					Issue: "Test file is in separate tests/ directory",
					Rule:  "Test files should be colocated with the code they test (location: colocated)",
					Fix:   "Move test file to the same directory as the code it tests",
				})
			}

		case "separate":
			// Test files should be in a tests/ directory
			if !strings.HasPrefix(relPath, "tests/") && !strings.Contains(relPath, "/tests/") {
				violations = append(violations, Violation{
					Type:  ViolationTestFileLocation,
					File:  relPath,
					Issue: "Test file is colocated with code instead of in tests/ directory",
					Rule:  "Test files should be in a separate tests/ directory (location: separate)",
					Fix:   "Move test file to tests/ directory mirroring the source structure",
				})
			}
		}
	}

	return violations
}

// validateBlackboxTests checks that all test files use blackbox testing (package name with _test suffix)
func (v *Validator) validateBlackboxTests() []Violation {
	var violations []Violation

	for _, node := range v.graph.GetNodes() {
		relPath := node.GetRelPath()
		packageName := node.GetPackage()

		// Only check test files
		if !strings.HasSuffix(relPath, "_test.go") {
			continue
		}

		// Check if this is a whitebox test (package without _test suffix)
		if !strings.HasSuffix(packageName, "_test") {
			// Determine the expected package name
			fileDir := filepath.Dir(relPath)
			fileDir = filepath.ToSlash(fileDir)

			// Get the base package name from the directory
			parts := strings.Split(fileDir, "/")
			basePkgName := parts[len(parts)-1]
			if basePkgName == "." {
				basePkgName = "main"
			}
			expectedPkg := basePkgName + "_test"

			violations = append(violations, Violation{
				Type:  ViolationWhiteboxTest,
				File:  relPath,
				Line:  1, // Package declaration is typically on line 1
				Issue: fmt.Sprintf("Test file uses whitebox testing (package %s instead of %s)", packageName, expectedPkg),
				Rule:  "Blackbox testing is enforced to ensure tests validate the public API, not internal implementation",
				Fix:   fmt.Sprintf("Change package declaration from 'package %s' to 'package %s'", packageName, expectedPkg),
			})
		}
	}

	return violations
}
