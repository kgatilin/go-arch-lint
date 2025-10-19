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

	if len(packages) == 0 {
		return results, nil
	}

	// Print header
	fmt.Printf("\n🔍 Running test coverage analysis for %d packages...\n\n", len(packages))

	for i, pkg := range packages {
		// Show progress
		fmt.Printf("  [%d/%d] Testing %s...", i+1, len(packages), getShortPackageName(pkg, r.moduleName))

		coverage, hasTests, err := r.runCoverageForPackage(pkg)
		if err != nil {
			// If coverage fails (e.g., no test files), record 0% with hasTests=false
			fmt.Printf(" no tests\n")
			results = append(results, PackageCoverage{
				PackagePath: pkg,
				Coverage:    0,
				hasTests:    false,
			})
			continue
		}

		if !hasTests {
			fmt.Printf(" no tests\n")
		} else {
			fmt.Printf(" %.1f%%\n", coverage)
		}

		results = append(results, PackageCoverage{
			PackagePath: pkg,
			Coverage:    coverage,
			hasTests:    hasTests,
		})
	}

	fmt.Println() // Empty line after progress

	return results, nil
}

// getShortPackageName extracts the relative package name from full import path
func getShortPackageName(pkgPath, moduleName string) string {
	if moduleName != "" && strings.HasPrefix(pkgPath, moduleName+"/") {
		return strings.TrimPrefix(pkgPath, moduleName+"/")
	}
	return pkgPath
}

// DirectorySummary represents coverage summary for a directory
type DirectorySummary struct {
	Directory      string
	PackageCount   int
	TestedPackages int
	TotalLines     int
	CoveredLines   int
	AvgCoverage    float64
}

// SummarizeBybDirectory groups coverage results by top-level directory
func SummarizeByDirectory(results []PackageCoverage, moduleName string, scanPaths []string) []DirectorySummary {
	// Group by directory
	dirStats := make(map[string]*DirectorySummary)

	for _, result := range results {
		// Get relative package path
		relPath := getShortPackageName(result.PackagePath, moduleName)

		// Extract top-level directory
		parts := strings.Split(relPath, "/")
		if len(parts) == 0 {
			continue
		}
		topDir := parts[0]

		// Initialize if not exists
		if _, exists := dirStats[topDir]; !exists {
			dirStats[topDir] = &DirectorySummary{
				Directory: topDir,
			}
		}

		stats := dirStats[topDir]
		stats.PackageCount++
		if result.hasTests {
			stats.TestedPackages++
		}
		// Accumulate coverage for averaging
		stats.AvgCoverage += result.Coverage
	}

	// Calculate averages and sort by directory name
	var summaries []DirectorySummary
	for dir, stats := range dirStats {
		if stats.PackageCount > 0 {
			stats.AvgCoverage = stats.AvgCoverage / float64(stats.PackageCount)
		}
		stats.Directory = dir
		summaries = append(summaries, *stats)
	}

	// Sort by directory name to match scan order
	sortedSummaries := make([]DirectorySummary, 0, len(scanPaths))
	for _, scanPath := range scanPaths {
		for _, summary := range summaries {
			if summary.Directory == scanPath {
				sortedSummaries = append(sortedSummaries, summary)
				break
			}
		}
	}

	// Add any directories not in scanPaths
	for _, summary := range summaries {
		found := false
		for _, existing := range sortedSummaries {
			if existing.Directory == summary.Directory {
				found = true
				break
			}
		}
		if !found {
			sortedSummaries = append(sortedSummaries, summary)
		}
	}

	return sortedSummaries
}

// PrintSummary displays a formatted coverage summary table
func PrintSummary(summaries []DirectorySummary, overallCoverage float64) {
	if len(summaries) == 0 {
		return
	}

	fmt.Println("📊 Coverage Summary by Directory:")
	fmt.Println()
	fmt.Println("┌────────────────────┬──────────┬─────────┬──────────────┐")
	fmt.Println("│ Directory          │ Packages │ Tested  │ Coverage     │")
	fmt.Println("├────────────────────┼──────────┼─────────┼──────────────┤")

	for _, summary := range summaries {
		coverageBar := getCoverageBar(summary.AvgCoverage)
		fmt.Printf("│ %-18s │ %8d │ %7d │ %5.1f%% %s │\n",
			truncate(summary.Directory, 18),
			summary.PackageCount,
			summary.TestedPackages,
			summary.AvgCoverage,
			coverageBar,
		)
	}

	fmt.Println("├────────────────────┴──────────┴─────────┼──────────────┤")
	overallBar := getCoverageBar(overallCoverage)
	fmt.Printf("│ Overall Project Coverage                │ %5.1f%% %s │\n", overallCoverage, overallBar)
	fmt.Println("└──────────────────────────────────────────┴──────────────┘")
	fmt.Println()
}

// getCoverageBar returns a visual bar representation of coverage
func getCoverageBar(coverage float64) string {
	barLength := 3
	filled := int((coverage / 100.0) * float64(barLength))
	if filled > barLength {
		filled = barLength
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < barLength; i++ {
		bar += "░"
	}

	return bar
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// CalculateOverallCoverage computes the overall project coverage
func CalculateOverallCoverage(results []PackageCoverage) float64 {
	if len(results) == 0 {
		return 0
	}

	totalCoverage := 0.0
	for _, result := range results {
		totalCoverage += result.Coverage
	}

	return totalCoverage / float64(len(results))
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
