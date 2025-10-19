package coverage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Config interface defines what coverage package needs from configuration
type Config interface {
	IsCoverageEnabled() bool
	GetCoverageThreshold() float64
	GetPackageThresholds() map[string]float64
}

// PackageCoverage represents test coverage for a single package
type PackageCoverage struct {
	PackagePath string
	Coverage    float64 // Percentage 0-100
	hasTests    bool    // Unexported to avoid conflict with HasTests() method
}

// GetPackagePath implements validator.PackageCoverage interface
func (pc PackageCoverage) GetPackagePath() string {
	return pc.PackagePath
}

// GetCoverage implements validator.PackageCoverage interface
func (pc PackageCoverage) GetCoverage() float64 {
	return pc.Coverage
}

// HasTests implements validator.PackageCoverage interface
func (pc PackageCoverage) HasTests() bool {
	return pc.hasTests
}

// Runner runs go test with coverage analysis
type Runner struct {
	projectPath string
	moduleName  string
}

// New creates a new coverage runner
func New(projectPath, moduleName string) *Runner {
	return &Runner{
		projectPath: projectPath,
		moduleName:  moduleName,
	}
}

// Run executes coverage analysis for all packages in scanPaths
func (r *Runner) Run(scanPaths []string) ([]PackageCoverage, error) {
	var results []PackageCoverage

	// Find all packages that should be analyzed
	packages, err := r.findPackages(scanPaths)
	if err != nil {
		return nil, fmt.Errorf("finding packages: %w", err)
	}

	for _, pkg := range packages {
		coverage, hasTests, err := r.runCoverageForPackage(pkg)
		if err != nil {
			// If coverage fails (e.g., no test files), record 0% with hasTests=false
			results = append(results, PackageCoverage{
				PackagePath: pkg,
				Coverage:    0,
				hasTests:    false,
			})
			continue
		}

		results = append(results, PackageCoverage{
			PackagePath: pkg,
			Coverage:    coverage,
			hasTests:    hasTests,
		})
	}

	return results, nil
}

// findPackages discovers all Go packages in the specified paths
func (r *Runner) findPackages(scanPaths []string) ([]string, error) {
	packagesMap := make(map[string]bool)

	for _, scanPath := range scanPaths {
		fullPath := filepath.Join(r.projectPath, scanPath)

		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip vendor and hidden directories
			if info.IsDir() && (info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".")) {
				return filepath.SkipDir
			}

			// Look for .go files (not test files, but any .go to indicate a package exists)
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
				pkgDir := filepath.Dir(path)
				relPath, err := filepath.Rel(r.projectPath, pkgDir)
				if err != nil {
					return err
				}

				// Convert to package import path
				pkgPath := filepath.ToSlash(relPath)
				if r.moduleName != "" {
					pkgPath = r.moduleName + "/" + pkgPath
				}

				packagesMap[pkgPath] = true
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	// Convert map to slice
	packages := make([]string, 0, len(packagesMap))
	for pkg := range packagesMap {
		packages = append(packages, pkg)
	}

	return packages, nil
}

// runCoverageForPackage runs go test -cover for a single package
func (r *Runner) runCoverageForPackage(pkgPath string) (float64, bool, error) {
	cmd := exec.Command("go", "test", "-cover", pkgPath)
	cmd.Dir = r.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if error is due to no test files
		if strings.Contains(string(output), "no test files") ||
			strings.Contains(string(output), "[no test files]") {
			return 0, false, nil
		}
		// Other errors (test failures, build errors) should still return coverage if available
		// Continue to parse output
	}

	// Parse coverage from output
	// Format: "coverage: 75.5% of statements"
	coverage, hasTests := parseCoverageOutput(string(output))
	return coverage, hasTests, nil
}

// parseCoverageOutput extracts coverage percentage from go test output
func parseCoverageOutput(output string) (float64, bool) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "coverage:") {
			// Extract percentage: "coverage: 75.5% of statements"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "coverage:" && i+1 < len(parts) {
					percentStr := strings.TrimSuffix(parts[i+1], "%")
					if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
						return percent, true
					}
				}
			}
		}
	}
	return 0, false
}

// GetThresholdForPackage determines the applicable threshold for a package
// using hierarchical inheritance (e.g., "cmd" applies to "cmd/foo/bar")
// pkgPath can be a full import path like "github.com/user/repo/cmd/foo" or relative like "cmd/foo"
func GetThresholdForPackage(pkgPath, moduleName string, defaultThreshold float64, packageThresholds map[string]float64) float64 {
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
