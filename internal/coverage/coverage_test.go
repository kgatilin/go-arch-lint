package coverage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kgatilin/go-arch-lint/internal/coverage"
)

func TestGetThresholdForPackage_Hierarchical(t *testing.T) {
	moduleName := "github.com/user/repo"

	tests := []struct {
		name              string
		pkgPath           string
		defaultThreshold  float64
		packageThresholds map[string]float64
		expected          float64
	}{
		{
			name:             "uses default when no package threshold",
			pkgPath:          "github.com/user/repo/foo/bar",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"cmd": 40,
			},
			expected: 70,
		},
		{
			name:             "uses exact match",
			pkgPath:          "github.com/user/repo/cmd",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"cmd": 40,
			},
			expected: 40,
		},
		{
			name:             "uses parent directory threshold",
			pkgPath:          "github.com/user/repo/cmd/foo/bar",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"cmd": 40,
			},
			expected: 40,
		},
		{
			name:             "uses most specific match",
			pkgPath:          "github.com/user/repo/internal/domain",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"internal":        80,
				"internal/domain": 90,
			},
			expected: 90,
		},
		{
			name:             "uses parent when child not defined",
			pkgPath:          "github.com/user/repo/internal/domain/user",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"internal":        80,
				"internal/domain": 90,
			},
			expected: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coverage.GetThresholdForPackage(tt.pkgPath, moduleName, tt.defaultThreshold, tt.packageThresholds)
			if result != tt.expected {
				t.Errorf("GetThresholdForPackage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPackageCoverage_InterfaceMethods(t *testing.T) {
	pc := coverage.PackageCoverage{
		PackagePath: "github.com/user/repo/pkg",
		Coverage:    75.5,
	}

	if pc.GetPackagePath() != "github.com/user/repo/pkg" {
		t.Errorf("GetPackagePath() = %v, want %v", pc.GetPackagePath(), "github.com/user/repo/pkg")
	}

	if pc.GetCoverage() != 75.5 {
		t.Errorf("GetCoverage() = %v, want %v", pc.GetCoverage(), 75.5)
	}
}

func TestRunner_Run_WithTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a package with code and tests
	pkgDir := filepath.Join(tmpDir, "pkg")
	err = os.MkdirAll(pkgDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create pkg directory: %v", err)
	}

	// Create a simple Go file
	goFile := `package pkg

func Add(a, b int) int {
	return a + b
}

func Multiply(a, b int) int {
	return a * b
}
`
	err = os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create Go file: %v", err)
	}

	// Create a test file that tests one function (partial coverage)
	testFile := `package pkg

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d, want 5", result)
	}
}
`
	err = os.WriteFile(filepath.Join(pkgDir, "main_test.go"), []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run coverage
	runner := coverage.New(tmpDir, "github.com/test/project")
	results, err := runner.Run([]string{"pkg"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should find one package
	if len(results) != 1 {
		t.Errorf("Run() found %d packages, want 1", len(results))
	}

	// Check that coverage was calculated
	if len(results) > 0 {
		result := results[0]
		if result.PackagePath != "github.com/test/project/pkg" {
			t.Errorf("PackagePath = %s, want github.com/test/project/pkg", result.PackagePath)
		}

		// Should have partial coverage (Add is tested, Multiply is not)
		if result.Coverage == 0 {
			t.Error("Expected non-zero coverage since tests exist")
		}

		if !result.HasTests() {
			t.Error("HasTests() = false, want true")
		}
	}
}

func TestRunner_Run_NoTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a package without tests
	pkgDir := filepath.Join(tmpDir, "internal")
	err = os.MkdirAll(pkgDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create internal directory: %v", err)
	}

	// Create a simple Go file
	goFile := `package internal

func Helper() string {
	return "helper"
}
`
	err = os.WriteFile(filepath.Join(pkgDir, "helper.go"), []byte(goFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create Go file: %v", err)
	}

	// Run coverage
	runner := coverage.New(tmpDir, "github.com/test/project")
	results, err := runner.Run([]string{"internal"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should find one package
	if len(results) != 1 {
		t.Errorf("Run() found %d packages, want 1", len(results))
	}

	// Check that coverage is 0
	if len(results) > 0 {
		result := results[0]
		if result.Coverage != 0 {
			t.Errorf("Coverage = %.1f, want 0", result.Coverage)
		}
		// Note: HasTests() behavior can vary depending on go test version
		// The important thing is coverage is 0
	}
}

func TestSummarizeByDirectory(t *testing.T) {
	results := []coverage.PackageCoverage{
		{PackagePath: "github.com/test/project/cmd", Coverage: 30.0},
		{PackagePath: "github.com/test/project/cmd/api", Coverage: 40.0},
		{PackagePath: "github.com/test/project/pkg/service", Coverage: 70.0},
		{PackagePath: "github.com/test/project/internal/domain", Coverage: 80.0},
	}

	summaries := coverage.SummarizeByDirectory(results, "github.com/test/project", []string{"cmd", "pkg", "internal"})

	// Should have 3 directories
	if len(summaries) != 3 {
		t.Errorf("SummarizeByDirectory() returned %d summaries, want 3", len(summaries))
	}

	// Check cmd summary (should average 30 and 40 = 35)
	cmdSummary := findSummary(summaries, "cmd")
	if cmdSummary == nil {
		t.Fatal("cmd summary not found")
	}
	if cmdSummary.PackageCount != 2 {
		t.Errorf("cmd PackageCount = %d, want 2", cmdSummary.PackageCount)
	}
	expectedAvg := (30.0 + 40.0) / 2.0
	if cmdSummary.AvgCoverage != expectedAvg {
		t.Errorf("cmd AvgCoverage = %.1f, want %.1f", cmdSummary.AvgCoverage, expectedAvg)
	}

	// Check pkg summary
	pkgSummary := findSummary(summaries, "pkg")
	if pkgSummary == nil {
		t.Fatal("pkg summary not found")
	}
	if pkgSummary.PackageCount != 1 {
		t.Errorf("pkg PackageCount = %d, want 1", pkgSummary.PackageCount)
	}
	if pkgSummary.AvgCoverage != 70.0 {
		t.Errorf("pkg AvgCoverage = %.1f, want 70.0", pkgSummary.AvgCoverage)
	}
}

func TestCalculateOverallCoverage(t *testing.T) {
	tests := []struct {
		name     string
		results  []coverage.PackageCoverage
		expected float64
	}{
		{
			name: "calculates average",
			results: []coverage.PackageCoverage{
				{Coverage: 50.0},
				{Coverage: 70.0},
				{Coverage: 80.0},
			},
			expected: (50.0 + 70.0 + 80.0) / 3.0,
		},
		{
			name:     "returns 0 for empty results",
			results:  []coverage.PackageCoverage{},
			expected: 0,
		},
		{
			name: "handles single package",
			results: []coverage.PackageCoverage{
				{Coverage: 75.5},
			},
			expected: 75.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coverage.CalculateOverallCoverage(tt.results)
			if result != tt.expected {
				t.Errorf("CalculateOverallCoverage() = %.1f, want %.1f", result, tt.expected)
			}
		})
	}
}

func TestRunner_Run_MultipleDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create cmd package
	cmdDir := filepath.Join(tmpDir, "cmd")
	err = os.MkdirAll(cmdDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create cmd directory: %v", err)
	}
	cmdFile := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(cmdFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create cmd file: %v", err)
	}

	// Create internal package
	internalDir := filepath.Join(tmpDir, "internal")
	err = os.MkdirAll(internalDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create internal directory: %v", err)
	}
	internalFile := `package internal

func Helper() {}
`
	err = os.WriteFile(filepath.Join(internalDir, "helper.go"), []byte(internalFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create internal file: %v", err)
	}

	// Run coverage on multiple directories
	runner := coverage.New(tmpDir, "github.com/test/project")
	results, err := runner.Run([]string{"cmd", "internal"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should find 2 packages
	if len(results) != 2 {
		t.Errorf("Run() found %d packages, want 2", len(results))
	}
}

func TestPrintSummary_EmptySummaries(t *testing.T) {
	// PrintSummary with empty summaries should return without printing
	// This tests the early return path
	var summaries []coverage.DirectorySummary
	coverage.PrintSummary(summaries, 0)
	// If no panic, test passes
}

func TestRunner_Run_NonExistentDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create cmd package (but NOT pkg)
	cmdDir := filepath.Join(tmpDir, "cmd")
	err = os.MkdirAll(cmdDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create cmd directory: %v", err)
	}
	cmdFile := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(cmdFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create cmd file: %v", err)
	}

	// Run coverage on cmd and pkg directories (pkg doesn't exist)
	runner := coverage.New(tmpDir, "github.com/test/project")
	results, err := runner.Run([]string{"cmd", "pkg", "internal"})

	// Should NOT return an error - should gracefully skip non-existent directories
	if err != nil {
		t.Fatalf("Run() should not error on non-existent directories, got error = %v", err)
	}

	// Should only find 1 package (cmd), not pkg or internal
	if len(results) != 1 {
		t.Errorf("Run() found %d packages, want 1 (only cmd exists)", len(results))
	}

	// Verify it found the cmd package
	if len(results) > 0 {
		if results[0].PackagePath != "github.com/test/project/cmd" {
			t.Errorf("Expected to find cmd package, got %s", results[0].PackagePath)
		}
	}
}

// Helper function to find a summary by directory name
func findSummary(summaries []coverage.DirectorySummary, dir string) *coverage.DirectorySummary {
	for i := range summaries {
		if summaries[i].Directory == dir {
			return &summaries[i]
		}
	}
	return nil
}
