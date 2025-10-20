package validator

import (
	"fmt"
	"strings"
)

// validateCoverage checks that test coverage meets configured thresholds
func (v *Validator) validateCoverage() []Violation {
	var violations []Violation

	defaultThreshold := v.cfg.GetCoverageThreshold()
	packageThresholds := v.cfg.GetPackageThresholds()
	moduleName := v.cfg.GetModule()

	for _, pkgCov := range v.coverageResults {
		pkgPath := pkgCov.GetPackagePath()
		coverage := pkgCov.GetCoverage()
		hasTests := pkgCov.HasTests()

		// Determine applicable threshold for this package (hierarchical)
		threshold := getThresholdForPackage(pkgPath, moduleName, defaultThreshold, packageThresholds)

		// Check if coverage is below threshold
		if coverage < threshold {
			var issue, fix string

			if !hasTests {
				issue = fmt.Sprintf("Package has no tests (0%% coverage, threshold: %.0f%%)", threshold)
				fix = fmt.Sprintf(`Add test files for this package:
1. Create %s_test.go files in the package directory
2. Write tests for the exported API
3. Run 'go test ./...' to verify`, pkgPath)
			} else {
				issue = fmt.Sprintf("Package coverage %.1f%% is below threshold %.0f%%", coverage, threshold)
				fix = fmt.Sprintf(`Improve test coverage for this package:
1. Run 'go test -cover %s' to see current coverage
2. Run 'go test -coverprofile=coverage.out %s && go tool cover -html=coverage.out' to see detailed coverage
3. Add tests for uncovered code paths
4. Consider table-driven tests for better coverage`, pkgPath, pkgPath)
			}

			violations = append(violations, Violation{
				Type:  ViolationLowCoverage,
				File:  pkgPath,
				Issue: issue,
				Rule:  fmt.Sprintf("Minimum test coverage: %.0f%% (hierarchical threshold)", threshold),
				Fix:   fix,
			})
		}
	}

	return violations
}

// getThresholdForPackage determines the applicable threshold for a package
// using hierarchical inheritance (e.g., "cmd" applies to "cmd/foo/bar")
// This duplicates logic from coverage.GetThresholdForPackage to maintain internal package isolation
func getThresholdForPackage(pkgPath string, moduleName string, defaultThreshold float64, packageThresholds map[string]float64) float64 {
	// Strip module prefix to get relative path
	// e.g., "github.com/user/repo/cmd/foo" -> "cmd/foo"
	relPath := pkgPath
	if moduleName != "" && strings.HasPrefix(pkgPath, moduleName+"/") {
		relPath = strings.TrimPrefix(pkgPath, moduleName+"/")
	} else if moduleName != "" && pkgPath == moduleName {
		// Package is the module root
		relPath = "."
	}

	// Start with default
	threshold := defaultThreshold

	// Find the most specific matching threshold
	// For package "cmd/foo/bar", check: "cmd/foo/bar", "cmd/foo", "cmd"
	parts := strings.Split(relPath, "/")

	// Check from most specific to least specific
	for i := len(parts); i > 0; i-- {
		prefix := strings.Join(parts[:i], "/")
		if t, exists := packageThresholds[prefix]; exists {
			threshold = t
			break
		}
	}

	return threshold
}
